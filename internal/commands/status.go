package commands

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dikee/hetzner-code/internal/hetzner"
	"github.com/dikee/hetzner-code/internal/state"
)

func age(createdAt string) string {
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return "?"
	}
	hours := time.Since(created).Hours()
	switch {
	case hours < 1:
		return fmt.Sprintf("%dm", int(hours*60))
	case hours < 24:
		return fmt.Sprintf("%.1fh", hours)
	default:
		return fmt.Sprintf("%.1fd", hours/24)
	}
}

func repoStr(repo state.Repo) string {
	branch := "default"
	if repo.Branch != nil && *repo.Branch != "" {
		branch = *repo.Branch
	}
	base := fmt.Sprintf("%s/%s@%s", repo.Owner, repo.Name, branch)
	if len(repo.Worktrees) > 0 {
		base += fmt.Sprintf(" [%s]", strings.Join(repo.Worktrees, ","))
	}
	return base
}

func reposStr(instance state.Instance) string {
	if len(instance.Repos) == 0 {
		return "(none)"
	}
	strs := make([]string, len(instance.Repos))
	for i, r := range instance.Repos {
		strs[i] = repoStr(r)
	}
	return strings.Join(strs, ", ")
}

// StatusOptions are every flag `hcode status` accepts.
type StatusOptions struct {
	Name       string
	JSONOutput bool
	Reconcile  bool
}

// Status runs `hcode status`.
func Status(opts StatusOptions) error {
	var instances []state.Instance
	if opts.Name != "" {
		instance, err := state.Load(opts.Name)
		if err != nil {
			return err
		}
		instances = []state.Instance{instance}
	} else {
		all, err := state.ListAll()
		if err != nil {
			return err
		}
		instances = all
	}

	if opts.JSONOutput {
		rows := make([]statusRow, len(instances))
		for i, inst := range instances {
			rows[i] = toRow(inst)
		}
		data, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else if len(instances) == 0 {
		fmt.Println("nothing tracked (see `hcode create`)")
	} else {
		printTable(instances)
	}

	if opts.Reconcile {
		return reconcile(instances)
	}
	return nil
}

type statusRow struct {
	Name     string   `json:"name"`
	IP       string   `json:"ip"`
	Type     string   `json:"type"`
	Location string   `json:"location"`
	Age      string   `json:"age"`
	Repos    []string `json:"repos"`
	OpsDir   *string  `json:"ops_dir"`
}

func toRow(instance state.Instance) statusRow {
	strs := make([]string, len(instance.Repos))
	for i, r := range instance.Repos {
		strs[i] = repoStr(r)
	}
	return statusRow{
		Name:     instance.Name,
		IP:       instance.IP,
		Type:     instance.Type,
		Location: instance.Location,
		Age:      age(instance.CreatedAt),
		Repos:    strs,
		OpsDir:   instance.OpsDir,
	}
}

func printTable(instances []state.Instance) {
	headers := []string{"NAME", "IP", "TYPE", "LOCATION", "AGE", "REPOS", "OPS"}
	rows := make([][]string, len(instances))
	for i, inst := range instances {
		opsDir := "-"
		if inst.OpsDir != nil {
			opsDir = *inst.OpsDir
		}
		rows[i] = []string{inst.Name, inst.IP, inst.Type, inst.Location, age(inst.CreatedAt), reposStr(inst), opsDir}
	}

	widths := make([]int, len(headers))
	for c, h := range headers {
		widths[c] = len(h)
	}
	for _, row := range rows {
		for c, cell := range row {
			if len(cell) > widths[c] {
				widths[c] = len(cell)
			}
		}
	}

	fmt.Println(joinPadded(headers, widths))
	for _, row := range rows {
		fmt.Println(joinPadded(row, widths))
	}
}

func joinPadded(cells []string, widths []int) string {
	padded := make([]string, len(cells))
	for i, c := range cells {
		padded[i] = c + strings.Repeat(" ", widths[i]-len(c))
	}
	return strings.Join(padded, "  ")
}

func reconcile(instances []state.Instance) error {
	tracked := map[string]bool{}
	for _, i := range instances {
		tracked[i.ServerID] = true
	}
	live, err := hetzner.ListManagedServers()
	if err != nil {
		return err
	}
	var orphans []hetzner.ServerInfo
	for _, s := range live {
		if !tracked[strconv.FormatInt(s.ID, 10)] {
			orphans = append(orphans, s)
		}
	}
	if len(orphans) > 0 {
		fmt.Println("\norphaned servers (hcode-labeled, not in local state):")
		for _, s := range orphans {
			fmt.Printf("  %s  id=%d  ip=%s\n", s.Name, s.ID, s.PublicNet.IPv4.IP)
		}
		fmt.Println("  clean up by hand: hcloud server delete <name>")
	}

	for _, instance := range instances {
		liveServer, err := hetzner.DescribeServer(instance.ServerID)
		if err != nil {
			return err
		}
		if liveServer == nil {
			fmt.Printf(
				"\n'%s' is tracked locally but no longer exists on Hetzner — its deploy "+
					"key(s) may still be live on GitHub. Run:\n"+
					"  hcode destroy %s --keep-key\nto clear local state, then remove the "+
					"deploy key(s) by hand if `gh` calls fail.\n",
				instance.Name, instance.Name,
			)
		}
	}
	return nil
}
