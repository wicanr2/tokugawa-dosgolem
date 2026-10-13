"""Docker 內以真實 X11 事件驗證桌面試作，所有子程序有界清理。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time


assert Path("/.dockerenv").exists() and os.getuid() == 1000
parser = argparse.ArgumentParser()
parser.add_argument("--mode", choices=["hd", "missing", "mouse"], default="hd")
args = parser.parse_args()
root = Path("/output")
assert root.stat().st_uid == os.getuid()
env = dict(os.environ, DISPLAY=":98", LIBGL_ALWAYS_SOFTWARE="1", LP_NUM_THREADS="2")
children = []


def run(*command, **kwargs):
    return subprocess.run(command, env=env, check=True, timeout=15, **kwargs)


def status():
    try:
        return json.loads((root / "desktop-status.json").read_text())
    except FileNotFoundError:
        return {}


def wait_for(predicate, limit=180):
    end = time.monotonic() + limit
    while time.monotonic() < end:
        s = status()
        if predicate(s):
            return s
        if process.poll() is not None:
            raise RuntimeError(f"desktop stopped early: {process.returncode}")
        time.sleep(0.05)
    raise TimeoutError(f"desktop condition timed out: {status()}")


def key(name):
    run("xdotool", "keydown", "--clearmodifiers", name)
    time.sleep(0.45)
    run("xdotool", "keyup", name)
    time.sleep(0.15)


def decoded(path):
    return run("ffmpeg", "-v", "error", "-threads", "1", "-i", str(path),
               "-f", "rawvideo", "-pix_fmt", "rgba", "-", stdout=subprocess.PIPE).stdout


def capture(label):
    previous = len(status().get("events") or [])
    key("F9")
    s = wait_for(lambda s: len(s.get("events") or []) > previous, 10)
    event = s["events"][-1]
    assert event["kind"] == "capture" and event["paused"]
    geometry = run("xdotool", "getwindowgeometry", "--shell", window, stdout=subprocess.PIPE).stdout.decode()
    g = dict(line.split("=", 1) for line in geometry.splitlines())
    actual = root / f"gui-{args.mode}-{label}.png"
    run("ffmpeg", "-v", "error", "-y", "-threads", "1", "-filter_threads", "1",
        "-f", "x11grab", "-video_size", f'{g["WIDTH"]}x{g["HEIGHT"]}',
        "-i", f':98+{g["X"]},{g["Y"]}', "-frames:v", "1", str(actual), stdout=subprocess.DEVNULL)
    expected = Path(event["frame"]["path"])
    a, b = decoded(actual), decoded(expected)
    if args.mode != "mouse":
        assert a == b, f"real window pixels differ from presented image: {actual}"
    return {"actual_png": str(actual), "sha256": hashlib.sha256(actual.read_bytes()).hexdigest(),
            "presented": event, "geometry": g, "window_pixels_equal": a == b}


try:
    xvfb_log = (root / f"xvfb-{args.mode}.log").open("wb")
    xvfb = subprocess.Popen(["Xvfb", ":98", "-screen", "0", "1600x1000x24", "-nolisten", "tcp", "-ac"],
                            env=env, stdout=xvfb_log, stderr=subprocess.STDOUT)
    children.append(xvfb)
    deadline = time.monotonic() + 10
    while not Path("/tmp/.X11-unix/X98").exists():
        assert xvfb.poll() is None and time.monotonic() < deadline
        time.sleep(0.05)
    for name in ("desktop-status.json", "desktop-status.pending"):
        (root / name).unlink(missing_ok=True)
    command = [str(root / "tokugawa-desktop-prototype"), "-desktop", "-desktop-speed", "20",
               "-steps", "150000000", "-hd-trace"]
    if args.mode != "mouse":
        source = Path("/project/workplace/cd-trace/first-play-input.json")
        assert hashlib.sha256(source.read_bytes()).hexdigest() == "cc23a9dc6353d066e12a1097224c1c8472940ade33365f880d01aff5ba6fe736"
        command += ["-mouse-file", str(source)]
    if args.mode == "missing":
        command += ["-hd-asset", "/output/nonexistent-hd-asset.png"]
    stdout_path = root / f"desktop-{args.mode}.json"
    stdout = stdout_path.open("wb")
    stderr = (root / f"desktop-{args.mode}.stderr.log").open("wb")
    process = subprocess.Popen(command, env=env, stdout=stdout, stderr=stderr)
    children.append(process)
    wait_for(lambda s: s.get("step", 0) > 1_000_000)
    window = run("xdotool", "search", "--name", "德川家康", stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
    run("xdotool", "windowfocus", "--sync", window)
    pictures = []
    if args.mode in ("hd", "missing"):
        wait_for(lambda s: s.get("eligible") and s.get("step", 0) >= 140_000_000)
        key("F6")
        wait_for(lambda s: s.get("enabled"), 10)
        key("F7")
        wait_for(lambda s: s.get("paused"), 10)
        pictures.append(capture("enabled"))
        key("F6")
        wait_for(lambda s: not s.get("enabled"), 10)
        pictures.append(capture("original"))
        key("F6")
        wait_for(lambda s: s.get("enabled"), 10)
        pictures.append(capture("restored"))
        assert pictures[0]["presented"]["frame"]["png_sha256"] == pictures[2]["presented"]["frame"]["png_sha256"]
        if args.mode == "hd":
            assert pictures[0]["presented"]["frame"]["mode"] == "hd"
            assert pictures[0]["presented"]["frame"]["png_sha256"] != pictures[1]["presented"]["frame"]["png_sha256"]
        else:
            assert pictures[0]["presented"]["frame"]["mode"] == "fallback_missing_asset"
            assert pictures[0]["presented"]["frame"]["png_sha256"] == pictures[1]["presented"]["frame"]["png_sha256"]
        key("F7")
    else:
        wait_for(lambda s: s.get("step", 0) >= 85_000_000)
        key("F7")
        wait_for(lambda s: s.get("paused"), 10)
        pictures.append(capture("menu"))
        run("xdotool", "windowsize", "--sync", window, "1000", "700")
        time.sleep(0.2)
        # 1280x800 畫布置中在 1000x700 視窗；點原版 (300,172)。
        run("xdotool", "mousemove", "--window", window, "469", "306")
        run("xdotool", "mousedown", "1")
        key("F7")
        s = wait_for(lambda s: not s.get("paused"), 10)
        start = s["step"]
        wait_for(lambda s: s.get("step", 0) >= start + 300_000, 10)
        run("xdotool", "mouseup", "1")
        wait_for(lambda s: s.get("step", 0) >= start + 5_000_000, 20)
        key("F7")
        wait_for(lambda s: s.get("paused"), 10)
        pictures.append(capture("after-click"))
        key("Escape")
    process.wait(timeout=60)
    assert process.returncode == 0
    stdout.close()
    stderr.close()
    receipt = json.loads(stdout_path.read_text())
    if args.mode != "mouse":
        assert receipt["successful_steps"] == 150_000_000 and receipt["stop"] == "instruction_budget"
        baseline = json.loads(Path("/project/workplace/hd-prototype/asset-capture/startup-probe/latest.json").read_text())
        fields = ["trace_tail", "interrupt_counts", "dpmi_calls", "dpmi_unimplemented", "frame", "cd_pcm",
                  "mounted_disc", "mouse_input_events", "input_boundary_frames"]
        assert all(receipt[k] == baseline[k] for k in fields), [k for k in fields if receipt[k] != baseline[k]]
        assert hashlib.sha256((root / "startup-memory.bin").read_bytes()).hexdigest() == "0d9123e52923ba852b125ea04918e1007138a33e0a8e4d856b078865cb33562f"
        toggles = [e for e in receipt["desktop"]["events"] if e["kind"] == "toggle"]
        assert len(toggles) == 3 and all(e["guest_unchanged"] for e in toggles)
        assert any(not e["paused"] for e in toggles)
        assert receipt["desktop"]["hd_observation"]["draw_completed"]
    else:
        assert receipt["stop"] == "desktop_closed" and receipt["desktop"]["physical_mouse_events"] > 0
        clicks = [e for e in receipt["mouse_input_events"] if e["buttons"] == 1]
        assert clicks and all(abs(e["x"] - 300) <= 1 and abs(e["y"] - 172) <= 1 for e in clicks), clicks
        assert receipt["mouse_input_events"][-1]["buttons"] == 0
        assert pictures[0]["presented"]["frame"]["png_sha256"] != pictures[1]["presented"]["frame"]["png_sha256"]
    result = {"mode": args.mode, "verified": True, "tool": "Xvfb + xdotool + ffmpeg; actual Ebitengine window",
              "receipt_sha256": hashlib.sha256(stdout_path.read_bytes()).hexdigest(), "pictures": pictures,
              "scope": "Linux container prototype; no physical audio or other OS verification"}
    (root / f"verification-{args.mode}.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
    print(f"desktop {args.mode}: verified", flush=True)
finally:
    for child in reversed(children):
        if child.poll() is None:
            child.terminate()
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait(timeout=5)
