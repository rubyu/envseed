FROM golang:1.25-bookworm

ARG TZ
ENV TZ="$TZ"

ARG CLAUDE_CODE_VERSION=latest
ARG CODEX_VERSION=latest
ARG NODE_VERSION=20

ENV GOPATH=/home/developer/go
ENV PATH=$PATH:$GOPATH/bin

ENV CLAUDE_CONFIG_DIR=/home/developer/.claude
ENV CODEX_CONFIG_DIR=/home/developer/.codex

# Install base packages and networking tools required by k3s/tailscale
RUN apt-get update && apt-get install -y --no-install-recommends \
    less \
    git \
    git-lfs \
    procps \
    sudo \
    fzf \
    zsh \
    man-db \
    locales \
    unzip \
    gnupg2 \
    gh \
    iptables \
    ipset \
    iproute2 \
    dnsutils \
    jq \
    nano \
    vim \
    curl \
    wget \
    ripgrep \
    build-essential \
    pass \
    bubblewrap \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Generate and configure en_US.UTF-8 locale
RUN sed -i 's/^# *en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen && \
    locale-gen en_US.UTF-8 && \
    update-locale LANG=en_US.UTF-8 LANGUAGE=en_US:en LC_ALL=en_US.UTF-8

ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Install Node.js for Claude Code
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - && \
    apt-get install -y nodejs && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Create a non-root user
ARG USERNAME=developer
ARG USER_UID=1000
ARG USER_GID=1000
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# Ensure NPM global directory exists and has correct permissions
RUN mkdir -p /usr/local/share/npm-global && \
    chown -R $USERNAME:$USERNAME /usr/local/share/npm-global

# Persist bash history
RUN SNIPPET="export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
    && mkdir /commandhistory \
    && touch /commandhistory/.bash_history \
    && chown -R $USERNAME /commandhistory

# Set DEVCONTAINER marker
ENV DEVCONTAINER=true

# Prepare workspace and config directories
RUN mkdir -p /workspace && \
    chown -R $USERNAME:$USERNAME /workspace && \
    mkdir -p $CLAUDE_CONFIG_DIR && \
    chown -R $USERNAME:$USERNAME $CLAUDE_CONFIG_DIR && \
    mkdir -p $CODEX_CONFIG_DIR && \
    chown -R $USERNAME:$USERNAME $CODEX_CONFIG_DIR && \
    mkdir -p /home/$USERNAME/go && \
    chown -R $USERNAME:$USERNAME /home/$USERNAME/go

# Install git-delta for better diffs
ARG GIT_DELTA_VERSION=0.18.2
RUN ARCH=$(dpkg --print-architecture) && \
    wget "https://github.com/dandavison/delta/releases/download/${GIT_DELTA_VERSION}/git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
    dpkg -i "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
    rm "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb"

# Copy helper scripts
COPY setup-firewall.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/setup-firewall.sh && \
    printf "%s\n%s\n" \
      "$USERNAME ALL=(root) NOPASSWD: /usr/local/bin/setup-firewall.sh" \
      > /etc/sudoers.d/$USERNAME-devtools && \
    chmod 0440 /etc/sudoers.d/$USERNAME-devtools

# Switch to non-root user
USER $USERNAME

# Ensure user has uv in PATH
ENV PATH="/home/$USERNAME/.local/bin:$PATH"
RUN curl -LsSf https://astral.sh/uv/install.sh | sh

# Install global NPM packages
ENV NPM_CONFIG_PREFIX=/usr/local/share/npm-global
ENV PATH=$PATH:/usr/local/share/npm-global/bin

# Set the default shell to zsh
ENV SHELL=/bin/zsh
ENV EDITOR=nano
ENV VISUAL=nano

# Install oh-my-zsh with plugins
ARG ZSH_IN_DOCKER_VERSION=1.2.0
RUN sh -c "$(wget -O- https://github.com/deluan/zsh-in-docker/releases/download/v${ZSH_IN_DOCKER_VERSION}/zsh-in-docker.sh)" -- \
    -p git \
    -p fzf \
    -p python \
    -p pip \
    -p golang \
    -a "source /usr/share/doc/fzf/examples/key-bindings.zsh" \
    -a "source /usr/share/doc/fzf/examples/completion.zsh" \
    -a "export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
    -x

# Install Go development tooling
RUN go install golang.org/x/tools/gopls@latest && \
    go install golang.org/x/tools/cmd/goimports@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest

# Ensure shell starts in workspace directory
RUN echo "cd /workspace" >> /home/$USERNAME/.zshrc

# Install Claude Code & OpenAI Codex CLIs
RUN npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}
RUN npm install -g @openai/codex@${CODEX_VERSION}

# Create cache directories with correct permissions
RUN mkdir -p /home/$USERNAME/.cache/uv && \
    chown -R $USERNAME:$USERNAME /home/$USERNAME/.cache

WORKDIR /workspace
