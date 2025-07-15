package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bilbo290/automagic/pkg/provider"
)

func SelectProject(projects []provider.Project) (*provider.Project, error) {
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

func SelectIssue(issues []provider.Issue) (*provider.Issue, error) {
	if len(issues) == 0 {
		return nil, fmt.Errorf("no issues available")
	}

	if len(issues) == 1 {
		fmt.Printf("Only one issue available: #%d %s\n", issues[0].IID, issues[0].Title)
		return &issues[0], nil
	}

	fmt.Printf("\nSelect an issue:\n")
	for i, issue := range issues {
		fmt.Printf("%d. #%d: %s\n", i+1, issue.IID, issue.Title)
		fmt.Printf("   State: %s\n", issue.State)
		if len(issue.Labels) > 0 {
			fmt.Printf("   Labels: %s\n", strings.Join(issue.Labels, ", "))
		}
		fmt.Printf("   Author: %s\n", issue.Author.Name)
		if issue.Assignee.Name != "" {
			fmt.Printf("   Assignee: %s\n", issue.Assignee.Name)
		}
		fmt.Printf("   Created: %s\n", issue.CreatedAt)
		fmt.Printf("   URL: %s\n\n", issue.WebURL)
	}

	fmt.Printf("Enter issue number (1-%d): ", len(issues))

	var choice int
	for {
		_, err := fmt.Scanf("%d", &choice)
		if err != nil {
			fmt.Printf("Invalid input. Please enter a number: ")
			continue
		}

		if choice < 1 || choice > len(issues) {
			fmt.Printf("Invalid choice. Please enter a number between 1 and %d: ", len(issues))
			continue
		}

		break
	}

	selected := &issues[choice-1]
	fmt.Printf("\nSelected issue: #%d %s\n", selected.IID, selected.Title)
	return selected, nil
}

func SelectLabelFilter() string {
	fmt.Printf("\nFilter issues by label:\n")
	fmt.Printf("1. All issues (no filter)\n")
	fmt.Printf("2. open\n")
	fmt.Printf("3. solved\n")
	fmt.Printf("4. picked_up_by_claude\n")
	fmt.Printf("Enter your choice (1-4): ")

	var choice int
	for {
		_, err := fmt.Scanf("%d", &choice)
		if err != nil {
			fmt.Printf("Invalid input. Please enter a number: ")
			continue
		}

		if choice < 1 || choice > 4 {
			fmt.Printf("Invalid choice. Please enter a number between 1 and 4: ")
			continue
		}

		break
	}

	switch choice {
	case 1:
		return ""
	case 2:
		return "open"
	case 3:
		return "solved"
	case 4:
		return "picked_up_by_claude"
	default:
		return ""
	}
}

func SelectOrganizations(providerInstance provider.Provider) ([]string, error) {
	orgs, err := providerInstance.GetUserOrganizations()
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %v", err)
	}

	if len(orgs) == 0 {
		fmt.Printf("No organizations found. Will list personal repositories only.\n")
		return []string{}, nil
	}

	fmt.Printf("\nSelect organizations to include (or press Enter for all):\n")
	fmt.Printf("0. [Personal] %s (your personal repositories)\n", providerInstance.GetUsername())
	for i, org := range orgs {
		fmt.Printf("%d. [Org] %s", i+1, org.Login)
		if org.Name != "" && org.Name != org.Login {
			fmt.Printf(" (%s)", org.Name)
		}
		if org.Description != "" {
			fmt.Printf(" - %s", org.Description)
		}
		fmt.Printf("\n")
	}
	fmt.Printf("%d. All organizations\n", len(orgs)+1)

	fmt.Printf("\nEnter organization numbers separated by commas (e.g., 0,1,3) or 'all': ")

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)

	if input == "" || input == "all" {
		selectedOrgs := []string{providerInstance.GetUsername()}
		for _, org := range orgs {
			selectedOrgs = append(selectedOrgs, org.Login)
		}
		return selectedOrgs, nil
	}

	parts := strings.Split(input, ",")
	selectedOrgs := make([]string, 0)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		choice, err := strconv.Atoi(part)
		if err != nil {
			fmt.Printf("Invalid input '%s', skipping\n", part)
			continue
		}

		if choice == 0 {
			selectedOrgs = append(selectedOrgs, providerInstance.GetUsername())
		} else if choice == len(orgs)+1 {
			selectedOrgs = []string{providerInstance.GetUsername()}
			for _, org := range orgs {
				selectedOrgs = append(selectedOrgs, org.Login)
			}
			break
		} else if choice > 0 && choice <= len(orgs) {
			selectedOrgs = append(selectedOrgs, orgs[choice-1].Login)
		} else {
			fmt.Printf("Invalid choice %d, skipping\n", choice)
		}
	}

	return selectedOrgs, nil
}