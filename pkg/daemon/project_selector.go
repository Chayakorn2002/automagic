package daemon

import (
	"fmt"

	"github.com/bilbo290/automagic/pkg/provider"
)

// selectProject handles interactive project selection for the daemon
func (d *Daemon) selectProject(projects []provider.Project) (*provider.Project, error) {
	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects available")
	}

	if len(projects) == 1 {
		fmt.Printf("Only one project available: %s\n", projects[0].PathWithNamespace)
		return &projects[0], nil
	}

	fmt.Printf("\nSelect a project:\n")
	for i, project := range projects {
		fmt.Printf("%d. %s\n", i+1, project.PathWithNamespace)
		if project.Description != "" {
			fmt.Printf("   Description: %s\n", project.Description)
		}
		fmt.Printf("   URL: %s\n", project.WebURL)
		fmt.Printf("   Visibility: %s\n\n", project.Visibility)
	}

	fmt.Printf("Enter project number (1-%d): ", len(projects))

	var choice int
	for {
		_, err := fmt.Scanf("%d", &choice)
		if err != nil {
			fmt.Printf("Invalid input. Please enter a number: ")
			continue
		}

		if choice < 1 || choice > len(projects) {
			fmt.Printf("Invalid choice. Please enter a number between 1 and %d: ", len(projects))
			continue
		}

		break
	}

	selected := &projects[choice-1]
	fmt.Printf("\nSelected project: %s\n", selected.PathWithNamespace)
	return selected, nil
}