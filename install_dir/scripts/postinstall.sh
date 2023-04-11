#!/bin/zsh
INSTALL_USER=$(ls -l /dev/console | awk '{print $3}')
SSH_CONFIG_DIR=/Users/$INSTALL_USER/.ssh
BLESS_CONFIG_DIR=/Users/$INSTALL_USER/.blessclient

echo "Fixing ownership of blessclient binary."
chown "$INSTALL_USER" "/usr/local/bin/blessclient"

if ! [ -d $BLESS_CONFIG_DIR ]; then
    echo "Creating Blessclient configuration folder."
    mkdir $BLESS_CONFIG_DIR
    chown -R "$INSTALL_USER" "$BLESS_CONFIG_DIR"
fi

# Move the SSH config to a backup
if [ -f "$BLESS_CONFIG_DIR/config.yml" ]; then
    echo "Copying the existing Blessclient config and replacing"
    mv "$BLESS_CONFIG_DIR/config.yml" "$BLESS_CONFIG_DIR/config.yml.bak"
fi

if ! [ -f "$BLESS_CONFIG_DIR/config.yml" ]; then
    echo "Moving example Blessclient config into place."
    mv "/tmp/blessclient-config.yml" "$BLESS_CONFIG_DIR/config.yml"
    echo "Updating the SSH config username."
    sed -i '' "s/REPLACEMEWITHUSERNAME/$INSTALL_USER/g" "$BLESS_CONFIG_DIR/config.yml"
    echo "Fixing ownership of blessclient config files"
    chown "$INSTALL_USER" "$BLESS_CONFIG_DIR/config.yml"
    rm -f "/tmp/blessclient-config.yml"
else
    echo "Blessclient config already exists, skipping."
fi

# Move the SSH config to a backup
if [ -f "$SSH_CONFIG_DIR/config" ]; then
    echo "Copying the existing SSH config and replacing"
    mv "$SSH_CONFIG_DIR/config" "$SSH_CONFIG_DIR/config.bak"
fi

# This ~/.ssh/ folder already exists from the preinstall script
if ! [ -f "$SSH_CONFIG_DIR/config" ]; then
    echo "Moving example SSH config into place."
    mv "/tmp/ssh-config" "$SSH_CONFIG_DIR/config"
    echo "Updating the SSH config username."
    sed -i '' "s/REPLACEMEWITHUSERNAME/$INSTALL_USER/g" "$SSH_CONFIG_DIR/config"
    echo "Fixing ownership of SSH config files"
    chown "$INSTALL_USER" "$SSH_CONFIG_DIR/config"
    rm -f "/tmp/ssh-config"
else
    echo "SSH config already exists, skipping."
fi

# This folder already exists from the preinstall script
if ! [ -f "$SSH_CONFIG_DIR/lyft_known_hosts" ]; then
    echo "Moving known hosts file into place."
    mv "/tmp/lyft_known_hosts" "$SSH_CONFIG_DIR/lyft_known_hosts"
    echo "Fixing ownership of SSH config files"
    chown "$INSTALL_USER" "$SSH_CONFIG_DIR/config"
    rm -f "/tmp/ssh-config"
else
    echo "SSH config already exists, skipping."
fi

# Make sure ownership for everything is all correct
echo "Fixing permissions on Bless and SSH config folders"
chown -R "$INSTALL_USER" "$BLESS_CONFIG_DIR"
chown -R "$INSTALL_USER" "$SSH_CONFIG_DIR"

echo "Fixing permissions on interaction script and adding to startup."
chown -R "$INSTALL_USER" "/opt/lyft/ssh_check_script.sh"
chmod +x "/opt/lyft/ssh_check_script.sh"
echo "/opt/lyft/ssh_check_script.sh" >> ~/.zshrc

