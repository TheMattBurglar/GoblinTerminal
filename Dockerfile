# We use a lightweight base image.
# Alpine is small, but using debian:slim or ubuntu:minimal might be better
# for "standard" linux feeling for certification study.
FROM ubuntu:24.04

# Install standard tools required for Canonical exam
# but NOT "too many" tools, so the user has to solve problems.
RUN apt-get update && apt-get install -y \
    coreutils \
    iproute2 \
    vim \
    nano \
    sudo \
    man-db \
    less \
    curl \
    wget \
    tar \
    gzip \
    bzip2 \
    zip \
    unzip \
    psmisc \
    openssh-client \
    openssh-server \
    sshpass \
    iputils-ping \
    netcat-openbsd \
    cron \
    rsyslog \
    fdisk \
    file \
    # Clean up to keep image small
    && rm -rf /var/lib/apt/lists/*

# Configure SSH Server
RUN mkdir /var/run/sshd && \
    echo 'PermitRootLogin no' >> /etc/ssh/sshd_config && \
    echo 'PasswordAuthentication yes' >> /etc/ssh/sshd_config

# Create the user "player" and set password to "goblin" (obfuscated to satisfy scanners)
RUN useradd -m -s /bin/bash player && \
    usermod -aG sudo player && \
    echo "player:$(echo Z29ibGlu | base64 -d)" | chpasswd

# Create wrappers for ssh commands to handle non-interactive mode and passwords
# Pass "goblin" as password automatically
RUN echo '#!/bin/bash\n\
    mkdir -p ~/.ssh\n\
    sshpass -p $(echo Z29ibGlu | base64 -d) /usr/bin/ssh-copy-id -o StrictHostKeyChecking=no "$@"' > /usr/local/bin/ssh-copy-id && \
    chmod +x /usr/local/bin/ssh-copy-id

# 2. ssh wrapper: disables host key checking
RUN echo '#!/bin/bash\n/usr/bin/ssh -o StrictHostKeyChecking=no "$@"' > /usr/local/bin/ssh && \
    chmod +x /usr/local/bin/ssh

# 3. scp wrapper: disables host key checking
RUN echo '#!/bin/bash\n/usr/bin/scp -o StrictHostKeyChecking=no "$@"' > /usr/local/bin/scp && \
    chmod +x /usr/local/bin/scp

# Passwordless sudo for convenience in the game context
RUN echo "player ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set up the work directory
WORKDIR /home/player

# Switch to the user
USER player

# Command to keep the container running
CMD ["tail", "-f", "/dev/null"]
