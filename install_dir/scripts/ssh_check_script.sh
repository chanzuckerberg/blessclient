#!/bin/zsh
echo "Your SSH configuration has been updated."
read -k 1 "REPLY?Would you like to test your Bless configuration? [y/n] "
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    #$(ssh -o BatchMode=yes -o ConnectTimeout=5 ssh gateway.ln echo ok 2>&1)
    echo "Testing. This may take a few seconds..."
    ssh -q gateway.ln exit
    if [[ $? == 0 ]]; then
        echo "Bless config test completed succesfully."
    else
        echo "Unsuccessful, Please troubleshoot using the following guide at https://confluence.lyft.net/pages/viewpage.action?pageId=220808108"
    fi
fi

read -k 1 "REPLY?Would you like to test your Github configuration? [y/n] "
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if ! [ -d ~/.ssh ]; then
        mkdir ~/.ssh
    fi

    if ! [ -f "$HOME/.ssh/id_rsa" ]; then
        cd ~/.ssh
        PASSWORD=$(openssl rand -base64 30)
        if [ "${#PASSWORD}" -ge 20 ]; then
            echo -n "Generating a key for you (may take a few seconds)..."
            ssh-keygen -b 8192 -P "$PASSWORD" -f id_rsa -o >/dev/null
            echo "done!"
            echo ""
            echo "echo $PASSWORD" > /tmp/askpass.sh
            chmod +x /tmp/askpass.sh
            export DISPLAY=":0"
            export SSH_ASKPASS="/tmp/askpass.sh"
            ssh-add -K "$HOME/.ssh/id_rsa" </dev/null
            rm -f /tmp/askpass.sh

        else
            err "Password generation failed in some way; did not obtain a valid password from openssl"
            return 1
        fi
    fi


    ssh-keygen -F github.com &> /dev/null
    if [ $? -ne 0 ]; then
        echo "Github.com is not a known SSH host, adding it (may take a few seconds)..."
        ssh-keyscan -T 5 github.com 2>&1 >> ~/.ssh/known_hosts
    fi

    git clone git@github.com:lyft/favoriteemoji.git >/dev/null 2>&1
    if [ -d favoriteemoji ]; then
        rm -rf favoriteemoji
        echo "Github config tests completed successfully"

    else
        echo "Unsuccesful, please setup your Github by following the guide at https://confluence.lyft.net/display/ENG/GitHub"
        exit 1
    fi
fi
echo "Completed, removing checks."
sed -i '' '/\/opt\/lyft\/ssh_check_script.sh/d' ~/.zshrc
exit 0

#rm -f /tmp/force_add.sh


#if ! [ -f "$SSH_CONFIG_DIR/id_rsa" ]; then
#    echo "Generating SSH private key."
#    cd $SSH_CONFIG_DIR
#    PASSWORD=$(openssl rand -base64 30)
#    if [ "${#PASSWORD}" -ge 20 ]; then
#        echo "Generating a key for you (may take a few seconds)..."
#        ssh-keygen -b 8192 -P "$PASSWORD" -f id_rsa -o -C $INSTALL_USER@lyft.com >/dev/null
#        echo "Owning SSH folder"
#        chown -R "$INSTALL_USER" $SSH_CONFIG_DIR
#        echo "echo $PASSWORD" > /tmp/askpass.sh
#        chmod +x /tmp/askpass.sh
#        chmod +x /tmp/force_add.sh
#        chown "$INSTALL_USER" /tmp/force_add.sh
        # We cannot forcibly add a key to a keychain
        # so instead we drop a script that will be run when zsh is started.
#        echo "export DISPLAY=\":0\"" > "/tmp/force_add.sh"
#        echo "export SSH_ASKPASS=\"/tmp/askpass.sh\"" >> "/tmp/force_add.sh"
#        echo "ssh-add --apple-use-keychain $SSH_CONFIG_DIR/id_rsa </dev/null"  >> "/tmp/force_add.sh"
#        echo "echo \"SSH setup has completed, you may close this terminal\"" >> "/tmp/force_add.sh"
#        echo "sed -i '' -e '$ d' ~/.zshrc" >> "/tmp/force_add.sh"
#        echo "exit" >> "/tmp/force_add.sh"
#        echo "/tmp/force_add.sh" >> ~/.zshrc
#        /bin/launchctl asuser $loggedInUID sudo -i -u $INSTALL_USER open /System/Applications/Utilities/Terminal.app
#    else
#        err "Password generation failed in some way; did not obtain a valid password from openssl"
#        return 1
#    fi
#else
#    echo "Existing id_rsa, skipping generation."
#fi
