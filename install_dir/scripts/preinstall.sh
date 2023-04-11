#!/bin/zsh
INSTALL_USER=$(ls -l /dev/console | awk '{print $3}')
SSH_CONFIG_DIR=/Users/$INSTALL_USER/.ssh
loggedInUID=$(id -u $INSTALL_USER)

if ! [ -d $SSH_CONFIG_DIR ]; then
    echo "Creating SSH configuration folder."
    mkdir $SSH_CONFIG_DIR
    chown -R "$INSTALL_USER" $SSH_CONFIG_DIR
    chmod 0700 $SSH_CONFIG_DIR
fi


if ! [ -f "$SSH_CONFIG_DIR/blessid" ]; then
    echo "Generating Bless SSH private key."
    if [[ ! -f $SSH_CONFIG_DIR/blessid ]]; then
        ssh-keygen -f $SSH_CONFIG_DIR/blessid -b 8192 -t rsa -C '' -N ''
    fi

    if [[ ! -f $SSH_CONFIG_DIR/blessid.pub ]]; then
        ssh-keygen -y -f $SSH_CONFIG_DIR/blessid > $SSH_CONFIG_DIR/blessid.pub
    fi

    # ssh looks for <identity>-cert before -cert.pub
    if [[ ! -L $SSH_CONFIG_DIR/blessid-cert ]]; then
        ln -s $SSH_CONFIG_DIR/blessid-cert.pub $SSH_CONFIG_DIR/blessid-cert
    fi
else
    echo "Existing bless key, skipping generation."
fi

exit 0
