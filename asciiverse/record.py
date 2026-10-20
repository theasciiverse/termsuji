#!/usr/bin/env python3
"""Records asciiverse/termsuji.cast, the catalog preview, at 100x30.

Usage: record.py <folder with termsuji and gnugo> <out.cast> [seconds]

asciinema rec runs `termsuji` (by name, from a throwaway folder on PATH,
with a throwaway home) in a detached 100x30 tmux session, with GNU Go
beside it. This script starts a new 9x9 game as Black against GNU Go at
termsuji's default strength (5), and plays Black's moves from a second
GNU Go of its own at strength 1. After each move it reads the game from
termsuji's save file ($XDG_DATA_HOME/termsuji/game.sgf) and replays GNU
Go's reply into its own engine. To place a stone it clears the cursor (q),
presses an arrow, which puts the cursor on the last stone played, walks to
the point and presses Return. It passes when its engine does, which ends
the game when GNU Go passes too; after `seconds` (default 45) it leaves
the game for the menu (the game stays saved) and quits."""
import os, re, shutil, subprocess, sys, tempfile, time

folder, out = os.path.abspath(sys.argv[1]), os.path.abspath(sys.argv[2])
secs = float(sys.argv[3]) if len(sys.argv) > 3 else 45.0

S = f"termsuji-preview-{os.getpid()}"
TMUX = ["tmux", "-L", S]
home = tempfile.mkdtemp()
bindir = os.path.join(home, "termsuji")
os.mkdir(bindir)
for name in ("termsuji", "gnugo"):
    shutil.copy(os.path.join(folder, name), os.path.join(bindir, name))
data = os.path.join(home, "data")
env = dict(os.environ, HOME=home, PATH=bindir + ":" + os.environ["PATH"],
           XDG_DATA_HOME=data, XDG_CONFIG_HOME=os.path.join(home, "config"),
           TERM="xterm-256color", COLORTERM="truecolor", LANG="C.UTF-8",
           USER="player", LOGNAME="player")
for k in ("VTE_VERSION", "TERM_PROGRAM", "TERM_PROGRAM_VERSION", "TMUX"):
    env.pop(k, None)
save = os.path.join(data, "termsuji", "game.sgf")
SIZE = 9
LETTERS = "ABCDEFGHJKLMNOPQRST"


def key(*k):
    subprocess.run(TMUX + ["send-keys", "-t", S, *k])


def running():
    return subprocess.run(TMUX + ["has-session", "-t", S], capture_output=True).returncode == 0


def screen():
    return subprocess.run(TMUX + ["capture-pane", "-p", "-t", S], capture_output=True, text=True).stdout


class GnuGo:
    """The driver's own GNU Go, for Black's moves."""
    def __init__(self, path):
        self.p = subprocess.Popen([path, "--mode", "gtp", "--level", "1"],
                                  stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
    def cmd(self, c):
        self.p.stdin.write(c + "\n"); self.p.stdin.flush()
        lines = []
        while True:
            line = self.p.stdout.readline()
            if line.strip() == "" and lines:
                break
            if line.strip():
                lines.append(line.strip())
        return " ".join(lines).lstrip("=? ").strip()


def moves():
    """The game's moves from termsuji's save: [(colour, x, y)], x -1 for a
    pass."""
    try:
        sgf = open(save).read()
    except OSError:
        return []
    out = []
    for c, v in re.findall(r";([BW])\[([a-s]{0,2})\]", sgf):
        if len(v) == 2:
            out.append((c, ord(v[0]) - 97, ord(v[1]) - 97))
        else:
            out.append((c, -1, -1))
    return out


def vertex(x, y):
    return f"{LETTERS[x]}{SIZE - y}"


def parse(v):
    v = v.upper()
    if v in ("PASS", "RESIGN"):
        return v
    return LETTERS.index(v[0]), SIZE - int(v[1:])


cmd = ("env TERM=xterm-256color asciinema rec --overwrite -q --window-size 100x30 --idle-time-limit 2"
       " --title termsuji --capture-env TERM,COLORTERM,LANG -c termsuji " + out)
subprocess.run(TMUX + ["new-session", "-d", "-s", S, "-x", "100", "-y", "30",
                       "-c", home, cmd], env=env, check=True)
gg = GnuGo(os.path.join(bindir, "gnugo"))
gg.cmd(f"boardsize {SIZE}"); gg.cmd("clear_board"); gg.cmd("komi 6.5")
try:
    time.sleep(2.0)
    # New game, the form's defaults (9x9, Black, strength 5): Tab to Play
    key("n"); time.sleep(1.2)
    for _ in range(4):
        key("Tab"); time.sleep(0.25)
    time.sleep(0.6); key("Enter"); time.sleep(1.5)
    t0 = time.time()
    played = 0
    while running() and time.time() - t0 < secs:
        if "The game is over" in screen():
            time.sleep(4.0)
            break
        ms = moves()
        if len(ms) % 2 == 1 or len(ms) < played:
            time.sleep(0.2)  # GNU Go has not answered yet
            continue
        # GNU Go's reply into the driver's engine
        for c, x, y in ms[played:]:
            gg.cmd(f"play {'black' if c == 'B' else 'white'} {'pass' if x < 0 else vertex(x, y)}")
        played = len(ms)
        mv = parse(gg.cmd("genmove black"))
        played += 1
        time.sleep(0.6)
        if mv in ("PASS", "RESIGN"):
            key("p"); time.sleep(1.0)
            continue
        tx, ty = mv
        # the cursor starts on the last move's stone, or the centre when
        # there is none or it was a pass
        last = (ms[-1][1], ms[-1][2]) if ms and ms[-1][1] >= 0 else (SIZE // 2, SIZE // 2)
        # q clears the cursor left from the last move; with no cursor it
        # would leave the game, so not before the first
        if ms:
            key("q"); time.sleep(0.15)
        key("Up"); time.sleep(0.25)
        cx, cy = last
        while (cx, cy) != (tx, ty):
            if cx < tx: key("Right"); cx += 1
            elif cx > tx: key("Left"); cx -= 1
            elif cy < ty: key("Down"); cy += 1
            else: key("Up"); cy -= 1
            time.sleep(0.12)
        time.sleep(0.4); key("Enter")
        # wait for termsuji to save Black's move
        for _ in range(50):
            if len(moves()) >= played:
                break
            time.sleep(0.1)
        time.sleep(0.5)
    time.sleep(1.0)
    # leave the game for the menu (it stays saved), then quit
    key("q"); time.sleep(0.3); key("q"); time.sleep(1.2); key("q")
    for _ in range(50):
        if not running():
            break
        time.sleep(0.1)
finally:
    gg.cmd("quit")
    subprocess.run(TMUX + ["kill-server"], capture_output=True)
    shutil.rmtree(home, ignore_errors=True)
