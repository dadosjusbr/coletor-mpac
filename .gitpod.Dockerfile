FROM gitpod/workspace-full

# Importante para executar o coletor localmente.

# Adding trusting keys to apt for repositories
RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | sudo apt-key add -
# Adding Google Chrome to the repositories
RUN sudo sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list'
# Updating apt to see and install Google Chrome
RUN sudo apt-get -y update
RUN sudo apt-get install -y google-chrome-stable