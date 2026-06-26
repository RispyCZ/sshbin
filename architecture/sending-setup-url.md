# Sending the setup URL

After a file upload completes, the SFTP server needs to tell the user where to go to finish configuring the share. There is no native SFTP protocol message for this, so we use a trick built into SSH itself.

## Mechanism

SSH multiplexes multiple channels over one connection. One channel type, `SSH_MSG_CHANNEL_EXTENDED_DATA` with data type `SSH_EXTENDED_DATA_STDERR`, is the remote process's stderr. Writing to it makes the text appear in the user's terminal even during an `scp` transfer.

charmbracelet/wish exposes this as `wish.WriteString(sess, msg)`, which writes to the session's stderr channel.

## Caveats

**It lands on stderr, not stdout.** The URL appears in the terminal but is not captured by shell substitution:
```sh
output=$(scp file.log sshbin.com)  # URL is NOT in $output
```
For a human reading their terminal this is fine.

**Client compatibility.** OpenSSH forwards remote stderr. Some other clients (WinSCP, certain libssh-based tools) silently discard it. Nothing functional should depend on the notice arriving — the setup URL is also available in the web UI under My Shares.
