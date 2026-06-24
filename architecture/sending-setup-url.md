# Setup URL

### How setup URL is shown to the user?

There to native way to print feedback after user uploads file to SFTP server over SSH. So we have to utilize simple trick and thats use stderr and SSH_MSG_CHANNEL_EXTENDED_DATA channel thats should be enaught to print user a URL where he can finish setup file sharing.

Two caveats:
- It lands on stderr, not stdout. It'll show on the terminal but won't be captured by something like output=$(scp ...). For a human-visible notice that's usually fine.
- This depends on the client forwarding remote stderr. OpenSSH does. Other clients (WinSCP, some libssh-based tools) may silently discard it, so don't make anything functional depend on the notice arriving.