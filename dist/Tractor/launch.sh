#!/bin/sh
# 拖拉机 · 升级 —— Miyoo Mini Flip launcher (stock firmware, direct framebuffer).
set -u

DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd) || exit 1
SAVE=${TRACTOR_SAVE_DIR:-/mnt/SDCARD/Saves/Tractor}
mkdir -p "$SAVE" 2>/dev/null || SAVE="$DIR/saves"
mkdir -p "$SAVE" 2>/dev/null || SAVE="$DIR"
export TRACTOR_SAVE_DIR="$SAVE"

LOG="$SAVE/launcher.log"
{
    echo "--- tractor $(date) ---"
    uname -a 2>/dev/null
    echo "fb: $(cat /proc/fb 2>/dev/null | tr '\n' ' ')"
    echo "pan: $(cat /sys/class/graphics/fb0/pan 2>/dev/null)"
    echo "mem: $(grep -E 'MemTotal|MemAvailable' /proc/meminfo 2>/dev/null | tr '\n' ' ')"
    echo "overcommit: $(cat /proc/sys/vm/overcommit_memory 2>/dev/null) ulimit-v: $(ulimit -v 2>/dev/null)"
    ls -l /dev/fb0 /dev/input/event* 2>/dev/null
} >>"$LOG" 2>&1

export GOGC=30
export GOMEMLIMIT=20MiB
ulimit -v unlimited 2>/dev/null || true
# Go reserves a large virtual heap; strict overcommit (2) refuses that mapping
# on a small-RAM handheld even though only a few megabytes are touched.
OLD_OC=$(cat /proc/sys/vm/overcommit_memory 2>/dev/null)
if [ "$OLD_OC" = "2" ]; then
    echo 1 > /proc/sys/vm/overcommit_memory 2>/dev/null && echo "[launcher] overcommit 2 -> 1" >>"$LOG"
fi
export LD_LIBRARY_PATH=/lib:/config/lib:/mnt/SDCARD/miyoo285/lib
cd "$DIR" || exit 1

case "$(uname -m)" in
    aarch64|arm64) BIN=./tractor-arm64 ;;
    *)             BIN=./tractor-arm ;;
esac

# Default: the panel is mounted 180 degrees against /dev/fb0.
# Create an empty file named rotate0 in the save folder if the picture is
# upside down on your unit.
ROT="--rotate180=true"
[ -f "$SAVE/rotate0" ] && ROT="--rotate180=false"

# Create an empty file named diag in the save folder to boot the colour bar /
# keypad diagnostic screen instead of the game.
MODE=""
[ -f "$SAVE/diag" ] && MODE="--diag"

echo "[launcher] $BIN $ROT $MODE" >>"$LOG"
"$BIN" $ROT $MODE >>"$LOG" 2>&1
echo "[launcher] exit $?" >>"$LOG"
[ "$OLD_OC" = "2" ] && echo 2 > /proc/sys/vm/overcommit_memory 2>/dev/null
sync
exit 0
