# Dev container base image.
#
# See https://github.com/devcontainers/images/tree/main/src/go for contents.
# This is the current `mcr.microsoft.com/devcontainers/go` image; the old
# `mcr.microsoft.com/vscode/devcontainers/go` namespace is deprecated.
#
# [Choice] Go version: 1, 1.25, 1.24 — append -bookworm/-trixie to pin the OS.
# Keep this at or above the `go` directive in go.mod (currently 1.25.8).
# GOTOOLCHAIN=auto will fetch the exact patch release if the image ships an
# older one, so the major.minor is what matters here.
ARG VARIANT="1.25-bookworm"
FROM mcr.microsoft.com/devcontainers/go:${VARIANT}

# [Choice] Node.js version: none, lts/*, 22, 20
ARG NODE_VERSION="none"
RUN if [ "${NODE_VERSION}" != "none" ]; then su vscode -c "umask 0002 && . /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# [Optional] Uncomment this section to install additional OS packages.
# RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
#     && apt-get -y install --no-install-recommends <your-package-list-here>

# [Optional] Uncomment the next lines to install additional Go tooling.
# USER vscode
# RUN go install <your-dependency-or-tool>@latest

# [Optional] Uncomment this line to install global node packages.
# RUN su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1
