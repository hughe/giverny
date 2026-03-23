#!/bin/sh
# Wrapper around diffreviewer that notifies outie to open a browser when
# diffreviewer starts listening.

REAL_DIFFREVIEWER="/usr/local/lib/giverny/diffreviewer"

# Create a temp fifo for captured stderr
stderr_fifo=$(mktemp -u /tmp/diffr-stderr.XXXXXX)
mkfifo "$stderr_fifo"

# Tee stderr: forward to the terminal and scan for the startup message.
(
    notified=false
    while IFS= read -r line; do
        printf '%s\n' "$line" >&2
        if [ "$notified" = false ] && printf '%s' "$line" | grep -q "DiffReviewer starting on"; then
            url=$(printf '%s' "$line" | grep -o 'http[^ ]*')
            if [ -n "$GIVERNY_CTRL_SOCK" ] && [ -n "$url" ]; then
                giverny --ctrl-send "OPEN-DIFFR $url" \
                    || printf 'Warning: failed to notify outie to open browser\n' >&2
            fi
            notified=true
        fi
    done
) < "$stderr_fifo" &
tee_pid=$!

# Run the real diffreviewer, redirecting its stderr into our fifo.
"$REAL_DIFFREVIEWER" "$@" 2>"$stderr_fifo"
exit_code=$?

wait $tee_pid 2>/dev/null
rm -f "$stderr_fifo"

exit $exit_code
