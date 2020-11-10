package github

import (
	"os"
	"strconv"
)

// https://docs.github.com/en/free-pro-team@latest/actions/reference/environment-variables#default-environment-variables

func githubEnv(name string) string {
	return os.Getenv("GITHUB_" + name)
}

func githubEnvNumber(name string) uint {
	i, err := strconv.ParseUint(githubEnv(name), 10, 64)
	if err != nil {
		return 0
	}
	return uint(i)
}

func withDefault(st ...string) string {
	for _, s := range st {
		if s != "" {
			return s
		}
	}
	return ""
}

// Workflow The name of the workflow being run
func Workflow() string {
	return githubEnv("WORKFLOW")
}

// RunID A unique number for each run within a repository. This number does not change if you re-run the workflow run.
func RunID() uint {
	return githubEnvNumber("RUN_ID")
}

// RunNumber A unique number for each run of a particular workflow in a repository.
//  This number begins at 1 for the workflow's first run, and increments with each new run.
//  This number does not change if you re-run the workflow run.
func RunNumber() uint {
	return githubEnvNumber("RUN_NUMBER")
}

// Action The unique identifier (id) of the action
func Action() string {
	return githubEnv("ACTION")
}

// Actions Always set to true when GitHub Actions is running the workflow.
// You can use this variable to differentiate when tests are being run locally or by GitHub Actions.
func Actions() bool {
	return githubEnv("ACTIONS") == "true"
}

// Actor The name of the person or app that initiated the workflow.
// For example, octocat.
func Actor() string {
	return githubEnv("ACTOR")
}

// Repository The owner and repository name.
// For example, octocat/Hello-World.
func Repository() string {
	return githubEnv("REPOSITORY")
}

// EventName The name of the webhook event that triggered the workflow.
func EventName() string {
	return githubEnv("EVENT_NAME")
}

// EventPath The path of the file with the complete webhook event payload.
// For example, /github/workflow/event.json.
func EventPath() string {
	return githubEnv("EVENT_PATH")
}

// Workspace The GitHub workspace directory path.
// The workspace directory is a copy of your repository if your workflow uses the actions/checkout action.
// If you don't use the actions/checkout action, the directory will be empty.
// For example, /home/runner/work/my-repo-name/my-repo-name.
func Workspace() string {
	return githubEnv("WORKSPACE")
}

// SHA The commit SHA that triggered the workflow.
// For example, ffac537e6cbbf934b08745a378932722df287a53.
func SHA() string {
	return githubEnv("SHA")
}

// Ref The branch or tag ref that triggered the workflow.
// For example, refs/heads/feature-branch-1. If neither a branch or tag is available for the event type, the variable will not exist.
func Ref() string {
	return githubEnv("REF")
}

// HeadRef Only set for forked repositories. The branch of the head repository.
func HeadRef() string {
	return githubEnv("HEAD_REF")
}

// BaseRef Only set for forked repositories. The branch of the base repository.
func BaseRef() string {
	return githubEnv("BASE_REF")
}

// ServerURL Returns the URL of the GitHub server.
// For example: https://github.com.
func ServerURL() string {
	return withDefault(githubEnv("SERVER_URL"), "https://github.com")
}

// APIURL Returns the API URL.
// For example: https://api.github.com.
func APIURL() string {
	return withDefault(githubEnv("API_URL"), "https://api.github.com")
}

// GraphQLURL Returns the GraphQL API URL.
// For example: https://api.github.com/graphql.
func GraphQLURL() string {
	return withDefault(githubEnv("API_URL"), "https://api.github.com/graphql")
}
