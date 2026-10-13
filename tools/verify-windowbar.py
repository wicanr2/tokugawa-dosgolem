"""Docker 內以普通 X11 事件驗證頂端功能列，子程序與等待均有界。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

assert Path("/.dockerenv").exists() and os.getuid() == 1000
parser = argparse.ArgumentParser()
parser.add_argument("--mode", choices=["hd", "mouse", "missing"], default="hd")
mode = parser.parse_args().mode
root = Path("/output")
assert root.stat().st_uid == 1000
env = dict(os.environ, DISPLAY=":98", LIBGL_ALWAYS_SOFTWARE="1", LP_NUM_THREADS="2")
children = []
pictures = []


def run(*command, **kwargs):
    return subprocess.run(command, env=env, check=True, timeout=15, **kwargs)


def status():
    try:
        return json.loads((root / "desktop-status.json").read_text())
    except FileNotFoundError:
        return {}


def wait_for(predicate, limit=15):
    deadline = time.monotonic() + limit
    while time.monotonic() < deadline:
        s = status()
        if predicate(s):
            return s
        assert process.poll() is None, (process.returncode, s)
        time.sleep(0.05)
    raise TimeoutError(status())


def key(name):
    run("xdotool", "keydown", "--clearmodifiers", name)
    time.sleep(0.4)
    run("xdotool", "keyup", name)
    time.sleep(0.15)


def move(x, y):
    run("xdotool", "mousemove", "--window", window, str(x), str(y))
    time.sleep(0.15)


def click(x, y):
    move(x, y)
    run("xdotool", "mousedown", "1")
    time.sleep(0.3)
    run("xdotool", "mouseup", "1")
    time.sleep(0.15)


def frame():
    count = len(status().get("events") or [])
    key("F9")
    s = wait_for(lambda s: len(s.get("events") or []) > count)
    e = s["events"][-1]
    assert e["kind"] == "capture"
    return e


def screenshot(label):
    geometry = run("xdotool", "getwindowgeometry", "--shell", window, stdout=subprocess.PIPE).stdout.decode()
    g = dict(line.split("=", 1) for line in geometry.splitlines())
    path = root / f"windowbar-{mode}-{label}.png"
    run("ffmpeg", "-v", "error", "-y", "-threads", "1", "-filter_threads", "1", "-f", "x11grab",
        "-draw_mouse", "0", "-video_size", f'{g["WIDTH"]}x{g["HEIGHT"]}', "-i", f':98+{g["X"]},{g["Y"]}',
        "-frames:v", "1", str(path), stdout=subprocess.DEVNULL)
    pictures.append({"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest(), "geometry": g})
    return path


def decoded(path):
    return run("ffmpeg", "-v", "error", "-threads", "1", "-i", str(path), "-f", "rawvideo",
               "-pix_fmt", "rgba", "-", stdout=subprocess.PIPE).stdout


try:
    log = (root / f"windowbar-{mode}-xvfb.log").open("wb")
    xvfb = subprocess.Popen(["Xvfb", ":98", "-screen", "0", "1600x1000x24", "-nolisten", "tcp", "-ac"],
                            env=env, stdout=log, stderr=subprocess.STDOUT)
    children.append(xvfb)
    deadline = time.monotonic() + 10
    while not Path("/tmp/.X11-unix/X98").exists():
        assert xvfb.poll() is None and time.monotonic() < deadline
        time.sleep(0.05)
    run("xdotool", "mousemove", "700", "500")
    for name in ("desktop-status.json", "desktop-status.pending"):
        (root / name).unlink(missing_ok=True)
    command = [str(root / "tokugawa-desktop-prototype"), "-desktop", "-desktop-speed", "20",
               "-steps", "150000000", "-hd-trace"]
    if mode != "mouse":
        source = Path("/project/workplace/cd-trace/first-play-input.json")
        assert hashlib.sha256(source.read_bytes()).hexdigest() == "cc23a9dc6353d066e12a1097224c1c8472940ade33365f880d01aff5ba6fe736"
        command += ["-mouse-file", str(source)]
    if mode == "missing":
        command += ["-hd-asset", "/output/no-hd.png"]
    stdout_path = root / f"windowbar-{mode}.json"
    stdout = stdout_path.open("wb")
    stderr = (root / f"windowbar-{mode}.stderr.log").open("wb")
    process = subprocess.Popen(command, env=env, stdout=stdout, stderr=stderr)
    children.append(process)
    wait_for(lambda s: s.get("step", 0) > 1000000, 60)
    window = run("xdotool", "search", "--name", "德川家康", stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
    run("xdotool", "windowfocus", "--sync", window)
    threshold = 85_000_000 if mode == "mouse" else 140_000_000
    wait_for(lambda s: s.get("step", 0) >= threshold, 180)
    key("F7")
    initial = wait_for(lambda s: s.get("paused"))
    assert not initial["bar_visible"]
    original = frame()
    move(400, 2)
    opened = wait_for(lambda s: s.get("bar_visible") and s["window_size"] == [1280, 864])
    assert not opened["bar_pinned"]
    frozen, mouse_count = opened["step"], opened["mouse_events"]
    click(150, 32)
    wait_for(lambda s: s["bar_menu"] == 0)
    move(150, 128)
    assert status()["bar_visible"] and status()["bar_menu"] == 0
    assert status()["step"] == frozen and status()["mouse_events"] == mouse_count
    dropdown = screenshot("display-menu")
    click(150, 128)
    wait_for(lambda s: s["bar_menu"] == -1)
    assert status()["enabled"] == (mode != "missing")
    assert status()["step"] == frozen
    move(400, 2)
    wait_for(lambda s: s["bar_visible"] and s["window_size"] == [1280, 864])
    chosen = frame()
    if mode != "mouse":
        assert chosen["proof"] == original["proof"]
    actual = screenshot("hd" if mode == "hd" else "bar")
    # 功能列占前 64 列；其下逐像素等於呈現層，原版未被壓縮。
    assert decoded(actual)[1280*64*4:] == decoded(Path(chosen["frame"]["path"]))
    move(900, 500)
    wait_for(lambda s: not s["bar_visible"] and s["window_size"] == [1280, 800])
    key("Escape")
    wait_for(lambda s: s["bar_visible"] and s["bar_pinned"])
    move(600, 406)
    click(600, 406)
    pinned = status()
    assert pinned["bar_visible"] and pinned["step"] == frozen
    pinned_mouse = pinned["mouse_events"]
    pinned_proof = frame()["proof"]
    # 鍵盤繼續，列仍固定，原版保持停止。
    key("Tab")
    key("Return")
    key("Up")
    key("Return")
    wait_for(lambda s: not s["paused"] and s["bar_visible"])
    time.sleep(0.3)
    assert status()["step"] == frozen and status()["mouse_events"] == pinned_mouse
    # 用滑鼠再暫停，工具選單擷取。
    click(500, 32)
    click(500, 128)
    wait_for(lambda s: s["paused"])
    click(900, 32)
    count = len(status().get("events") or [])
    click(900, 88)
    wait_for(lambda s: len(s.get("events") or []) > count)
    assert status()["events"][-1]["kind"] == "capture"
    assert status()["events"][-1]["proof"] == pinned_proof
    # 叉號按住後移到遊戲區，不得把該次按下轉送原版。
    key("F7")
    wait_for(lambda s: not s["paused"] and s["bar_visible"])
    move(1230, 32)
    run("xdotool", "mousedown", "1")
    wait_for(lambda s: not s["bar_visible"])
    move(600, 342)
    assert status()["mouse_events"] == pinned_mouse and status()["mouse_blocked"]
    assert status()["step"] == frozen, "closing gesture must stay fenced until release"
    key("F7")
    wait_for(lambda s: s["paused"])
    run("xdotool", "mouseup", "1")
    wait_for(lambda s: not s["mouse_blocked"])
    move(400, 2)
    wait_for(lambda s: s["bar_visible"])
    key("Escape")
    wait_for(lambda s: not s["bar_visible"])
    time.sleep(0.3)
    assert not status()["bar_visible"], "escape close must not immediately reopen at top"
    move(900, 500)
    move(400, 2)
    wait_for(lambda s: s["bar_visible"])
    move(900, 500)
    wait_for(lambda s: not s["bar_visible"])
    # 縮放視窗開關功能列，遊戲比例與原尺寸一致。
    run("xdotool", "windowsize", "--sync", window, "1000", "700")
    wait_for(lambda s: s["window_size"] == [1000, 700])
    key("Escape")
    wait_for(lambda s: s["bar_visible"] and s["window_size"] == [1000, 750])
    screenshot("scaled")
    key("Escape")
    wait_for(lambda s: not s["bar_visible"] and s["window_size"] == [1000, 700])
    if mode == "mouse":
        before_click = frame()
        move(469, 306)
        run("xdotool", "mousedown", "1")
        key("F7")
        start = wait_for(lambda s: not s["paused"])["step"]
        wait_for(lambda s: s["step"] >= start + 300_000)
        run("xdotool", "mouseup", "1")
        wait_for(lambda s: s["step"] >= start + 5_000_000)
        key("F7")
        after_click = frame()
        assert before_click["frame"]["png_sha256"] != after_click["frame"]["png_sha256"]
        screenshot("new-game")
        key("shift+Escape")
    else:
        key("F7")
    process.wait(timeout=60)
    assert process.returncode == 0
    stdout.close()
    receipt = json.loads(stdout_path.read_text())
    options = [e for e in receipt["desktop"]["windowbar_events"] if e["kind"] == "option"]
    assert len(options) >= (3 if mode == "missing" else 4) and all(e["guest_unchanged"] for e in options)
    if mode != "mouse":
        baseline = json.loads(Path("/project/workplace/hd-prototype/asset-capture/startup-probe/latest.json").read_text())
        fields = ["successful_steps", "trace_tail", "interrupt_counts", "dpmi_calls", "frame", "cd_pcm",
                  "mounted_disc", "mouse_input_events", "input_boundary_frames"]
        assert all(receipt[k] == baseline[k] for k in fields), [k for k in fields if receipt[k] != baseline[k]]
        assert hashlib.sha256((root / "startup-memory.bin").read_bytes()).hexdigest() == "0d9123e52923ba852b125ea04918e1007138a33e0a8e4d856b078865cb33562f"
    else:
        clicks = [e for e in receipt["mouse_input_events"] if e["buttons"] == 1]
        assert clicks and all(abs(e["x"] - 300) <= 1 and abs(e["y"] - 172) <= 1 for e in clicks), clicks
        assert receipt["mouse_input_events"][-1]["buttons"] == 0
    result = {"mode": mode, "verified": True, "tool": "ordinary X11 events, Xvfb + xdotool + ffmpeg",
              "receipt_sha256": hashlib.sha256(stdout_path.read_bytes()).hexdigest(), "pictures": pictures,
              "frozen_step": frozen, "menu_options": len(options), "game_pixels_equal_below_bar": True,
              "scope": "Linux container; no hardware mouse or audio listening assertion"}
    (root / f"verification-windowbar-{mode}.json").write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(f"window bar {mode}: verified", flush=True)
finally:
    for child in reversed(children):
        if child.poll() is None:
            child.terminate()
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait(timeout=5)
