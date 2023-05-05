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
            echo "echo $PASSWORD" > $HOME/.ssh/askpass.sh
            chmod +x $HOME/.ssh/askpass.sh
            export DISPLAY=":0"
            export SSH_ASKPASS="$HOME/.ssh/askpass.sh"
            ssh-add --apple-use-keychain "$HOME/.ssh/id_rsa" </dev/null
            #rm -f ~/.ssh/askpass.sh

        else
            err "Password generation failed in some way; did not obtain a valid password from openssl"
            return 1
        fi
        echo "Your SSH key has been generated, please do the following:"
        echo "1) Click on the following link to navigate to your Github SSH settings - https://github.com/settings/ssh"
        echo "2) select 'New SSH Key' button"
        echo "3) Add a 'title' such as 'Lyft SSH Key"
        echo "4) Paste the contents of `~/.ssh/id_rsa.pub` into the 'key' textbox (This should already be copied into your clipboard)"
        pbcopy < $HOME/.ssh/id_rsa.pub
        echo "5) Once added, next to the SSH key select 'Configure SSO' and select 'authorize' next to the Lyft organization and follow the prompts."
        echo "If you run into any issues, check out  https://confluence.lyft.net/display/ENG/GitHub"
        echo "When you've completed this, press any key to continue... "
        read -k 1 ""
        #echo "3) Click the 'Authorize' button next to your new SSH key
        #echo "1) Note: Your SSH key has been copied to your clipboard."

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
