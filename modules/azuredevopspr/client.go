package azuredevopspr

import (
	"github.com/google/uuid"
	azrGit "github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/pkg/errors"
)

func (widget *Widget) getPullRequests(project string) ([]azrGit.GitPullRequest, error) {
	var pullRequests []azrGit.GitPullRequest
	top := widget.settings.maxRows

	//u := uuid.MustParse("e2954445-077d-67dc-89a6-32d7d4f01c92")
	u := uuid.MustParse(widget.settings.userUuid)

	prs, err := widget.git.GetPullRequestsByProject(widget.ctx,
		azrGit.GetPullRequestsByProjectArgs{Project: &project, Top: &top,
			SearchCriteria: &azrGit.GitPullRequestSearchCriteria{Status: &azrGit.PullRequestStatusValues.Active,
				CreatorId: &u}})

	if err != nil {
		return pullRequests, errors.Wrap(err, "could not get prs")
	}

	return *prs, nil
}
