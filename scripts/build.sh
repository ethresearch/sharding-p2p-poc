GIT_COMMIT=$(git rev-list -1 HEAD) && go build -ldflags "-X main.GitCommit=$GIT_COMMIT"
