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
parser.add_argument("--mode", choices=["hd", "missing", "mouse", "scroll", "scroll-missing", "scroll-only", "portraits", "portraits-missing", "frame", "frame-missing", "frame-only", "window", "window-missing", "window-only", "window-modal", "terrain", "terrain-missing"], default="hd")
parser.add_argument("--include-council", action="store_true", help="於既有普通輸入路徑另抽測 GUNGI.OZM")
args = parser.parse_args()
root = Path("/output")
assert root.stat().st_uid == os.getuid()
(root / "startup-probe").mkdir(exist_ok=True)
assert (root / "startup-probe").stat().st_uid == os.getuid()
env = dict(os.environ, DISPLAY=":98", LIBGL_ALWAYS_SOFTWARE="1", LP_NUM_THREADS="2")
children = []
scroll_checks = []
is_scroll = args.mode.startswith("scroll")
is_portraits = args.mode.startswith("portraits")
is_terrain = args.mode.startswith("terrain")
long_path = is_portraits or is_terrain
terrain_checks = []
assert not args.include_council or is_portraits
story_cases = 9 if args.include_council else 8
portrait_checks = []
kao_checks = []
is_frame = args.mode.startswith("frame")
frame_checks = []
is_window = args.mode.startswith("window")
window_checks = []


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
    before = status()
    run("xdotool", "keydown", "--clearmodifiers", name)
    try:
        # 切換會同步產生完整狀態雜湊與 PNG；等實際回應後才放開。
        # 固定短暫按鍵可能整段落在上一個仍未完成的 Update 內。
        if name == "F6":
            wait_for(lambda s: s.get("enabled") != before.get("enabled"), 15)
        elif name == "F7":
            wait_for(lambda s: s.get("paused") != before.get("paused"), 15)
        elif name == "F9":
            wait_for(lambda s: len(s.get("events") or []) > len(before.get("events") or []) and s["events"][-1]["kind"] == "capture", 15)
        else:
            time.sleep(0.45)
    finally:
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


def check_scroll_pair(enhanced, original):
    a, b = decoded(Path(enhanced["actual_png"])), decoded(Path(original["actual_png"]))
    source = decoded(Path("/project/workplace/hd-prototype/scroll/scroll-original.png"))
    assert len(a) == len(b) == 1280*800*4 and len(source) == 640*150*4
    assert enhanced["presented"]["proof"] == original["presented"]["proof"]
    foreground = differences = 0
    for y in range(800):
        for x in range(1280):
            i = (y*1280+x)*4
            p, q = a[i:i+4], b[i:i+4]
            portrait = 320 <= x < 960 and 100 <= y < 400
            if y < 500 and (not portrait or args.mode == "scroll-only"):
                assert p == q, ("outside HD region", x, y)
            if y >= 500:
                j = (((y-500)//2)*640+x//2)*4
                if q != source[j:j+4]:
                    assert p == q, ("original foreground changed", x, y)
                    foreground += 1
                if p != q:
                    differences += 1
    assert foreground > 0
    if args.mode == "scroll-missing":
        assert differences == 0, "missing scroll must keep original while portrait stays HD"
    else:
        assert differences > 0
    scroll_checks.append({"step": enhanced["presented"]["step"], "foreground_output_pixels": foreground,
                          "changed_scroll_pixels": differences, "foreground_unchanged": True,
                          "outside_regions_unchanged": True})


def check_window_modal(label):
    s = status()
    rects = [(d["logical_rect"]["Min"]["X"], d["logical_rect"]["Min"]["Y"],
              d["logical_rect"]["Max"]["X"], d["logical_rect"]["Max"]["Y"]) for d in s["window_draws"]]
    modes = s["window_modes"]
    assert "hd" in modes and s["kao_hd_count"] == 0, s
    enhanced = capture(label + "-enabled")
    key("F6")
    original = capture(label + "-original")
    key("F6")
    restored = capture(label + "-restored")
    assert enhanced["presented"]["proof"] == original["presented"]["proof"]
    assert enhanced["presented"]["frame"]["png_sha256"] == restored["presented"]["frame"]["png_sha256"]
    a, b = decoded(Path(enhanced["actual_png"])), decoded(Path(original["actual_png"]))
    source = decoded(Path("/project/workplace/hd-prototype/catalog/window/WINDOW-original.png"))
    changed = foreground = outside = 0
    for y in range(800):
        for x in range(1280):
            i = (y*1280+x)*4
            p, q = a[i:i+4], b[i:i+4]
            n = next((n for n in reversed(range(len(rects))) if rects[n][0] <= x//2 < rects[n][2] and rects[n][1] <= y//2 < rects[n][3]), None)
            if n is None or modes[n] != "hd":
                assert p == q, (label, "original fallback or outside window changed", x, y)
                outside += 1
                continue
            rect = rects[n]
            lx, ly, w, h = x//2-rect[0], y//2-rect[1], rect[2]-rect[0], rect[3]-rect[1]
            col = 0 if lx < 16 else 2 if lx >= w-16 else 1
            row = 0 if ly < 16 else 2 if ly >= h-16 else 1
            j = ((ly%16)*144+(row*3+col)*16+lx%16)*4
            if q != source[j:j+4]:
                assert p == q, (label, "original modal foreground changed", x, y)
                foreground += 1
            changed += p != q
    assert changed > 0 and foreground > 0 and outside > 0
    window_checks.append({"label": label, "step": s["step"], "rects": rects, "modes": modes,
                          "changed_background_pixels": changed, "preserved_foreground_pixels": foreground,
                          "preserved_outside_or_fallback_pixels": outside, "guest_unchanged": True})
    return [enhanced, original, restored]


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
    command = [str(root / "tokugawa-desktop-prototype"), "-desktop", "-desktop-speed", "5" if is_scroll else "60" if long_path or is_frame or is_window else "20",
               "-steps", "750000000" if long_path else "150000000", "-hd-trace"]
    if args.mode != "mouse":
        source = Path("/project/tools/player-path.json") if long_path else Path("/project/workplace/cd-trace/first-play-input.json")
        if long_path:
            baseline = json.loads(Path("/project/workplace/desktop-prototype/headless/startup-probe/latest.json").read_text())
            assert hashlib.sha256(source.read_bytes()).hexdigest() == baseline["mouse_input_sha256"]
        else:
            assert hashlib.sha256(source.read_bytes()).hexdigest() == "cc23a9dc6353d066e12a1097224c1c8472940ade33365f880d01aff5ba6fe736"
        command += ["-mouse-file", str(source)]
    if args.mode == "missing":
        command += ["-hd-asset", "/output/nonexistent-hd-asset.png", "-hd-scroll-asset", "/output/nonexistent-scroll.png", "-hd-portrait-dir", "/output/nonexistent-portrait-directory", "-hd-kao-dir", "/output/nonexistent-kao-directory"]
    if args.mode == "scroll-missing":
        command += ["-hd-scroll-asset", "/output/nonexistent-scroll.png"]
    if args.mode == "scroll-only":
        command += ["-hd-asset", "/output/nonexistent-hd-asset.png"]
    if args.mode == "portraits-missing":
        command += ["-hd-portrait-dir", "/output/nonexistent-portrait-directory", "-hd-kao-dir", "/output/nonexistent-kao-directory"]
    if not is_frame or args.mode == "frame-missing":
        command += ["-hd-frame-asset", "/output/nonexistent-frame.png"]
    if args.mode == "frame-only":
        command += ["-hd-asset", "/output/no-story.png", "-hd-portrait-dir", "/output/no-stories", "-hd-kao-dir", "/output/no-heads", "-hd-scroll-asset", "/output/no-scroll.png"]
    if not is_window or args.mode == "window-missing":
        command += ["-hd-window-asset", "/output/no-window.png"]
    if args.mode in ("window-only", "window-modal"):
        command += ["-hd-asset", "/output/no-story.png", "-hd-portrait-dir", "/output/no-stories", "-hd-kao-dir", "/output/no-heads", "-hd-scroll-asset", "/output/no-scroll.png"]
    if is_terrain:
        command += ["-hd-asset", "/output/no-story.png", "-hd-portrait-dir", "/output/no-stories", "-hd-kao-dir", "/output/no-heads", "-hd-scroll-asset", "/output/no-scroll.png"]
        if args.mode == "terrain-missing":
            command += ["-hd-terrain-dir", "/output/no-terrain"]
    stdout_path = root / f"desktop-{args.mode}.json"
    stdout = stdout_path.open("wb")
    stderr = (root / f"desktop-{args.mode}.stderr.log").open("wb")
    process = subprocess.Popen(command, env=env, stdout=stdout, stderr=stderr)
    children.append(process)
    wait_for(lambda s: s.get("step", 0) > 1_000_000)
    window = run("xdotool", "search", "--name", "德川家康", stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
    run("xdotool", "windowfocus", "--sync", window)
    pictures = []
    if is_window:
        wait_for(lambda s: len(s.get("window_draws") or []) >= 5 and s.get("step", 0) >= 100_000_000)
        key("F6")
        key("F7")
        s = status()
        assert s["step"] < 105_100_000, "sample must precede the normal confirmation click"
        rects = [(d["logical_rect"]["Min"]["X"], d["logical_rect"]["Min"]["Y"],
                  d["logical_rect"]["Max"]["X"], d["logical_rect"]["Max"]["Y"]) for d in s["window_draws"]]
        expected = "fallback_missing_asset" if args.mode == "window-missing" else "hd"
        assert s["window_modes"] == [expected]*len(rects), s
        assert (s["kao_hd_count"] == 4) == (args.mode not in ("window-only", "window-modal")), s
        enhanced = capture("enabled")
        key("F6")
        original = capture("original")
        key("F6")
        restored = capture("restored")
        assert enhanced["presented"]["proof"] == original["presented"]["proof"]
        assert enhanced["presented"]["frame"]["png_sha256"] == restored["presented"]["frame"]["png_sha256"]
        a, b = decoded(Path(enhanced["actual_png"])), decoded(Path(original["actual_png"]))
        source = decoded(Path("/project/workplace/hd-prototype/catalog/window/WINDOW-original.png"))
        changed = foreground = outside = 0
        for y in range(800):
            for x in range(1280):
                i = (y*1280+x)*4
                p, q = a[i:i+4], b[i:i+4]
                rect = next((r for r in reversed(rects) if r[0] <= x//2 < r[2] and r[1] <= y//2 < r[3]), None)
                head = any((96+k*112)*2 <= x < (176+k*112)*2 and 172 <= y < 332 for k in range(4))
                if rect is None:
                    assert p == q, ("outside original windows", x, y)
                    outside += 1
                    continue
                if head and args.mode not in ("window-only", "window-modal"):
                    continue
                lx, ly, w, h = x//2-rect[0], y//2-rect[1], rect[2]-rect[0], rect[3]-rect[1]
                col = 0 if lx < 16 else 2 if lx >= w-16 else 1
                row = 0 if ly < 16 else 2 if ly >= h-16 else 1
                j = ((ly%16)*144+(row*3+col)*16+lx%16)*4
                if q != source[j:j+4]:
                    assert p == q, ("window original foreground changed", x, y)
                    foreground += 1
                changed += p != q
        assert (changed > 0) == (expected == "hd"), (changed, expected)
        assert foreground > 0 and outside > 0
        window_checks.append({"step": s["step"], "rects": rects, "mode": expected,
                              "changed_background_pixels": changed, "preserved_foreground_pixels": foreground,
                              "preserved_outside_pixels": outside, "kao_hd_count": s["kao_hd_count"], "guest_unchanged": True})
        pictures += [enhanced, original, restored]
        key("F7")
        if args.mode == "window-modal":
            wait_for(lambda s: s.get("step", 0) >= 107_000_000)
            key("F7")
            assert any(d["logical_rect"] == {"Min": {"X": 200, "Y": 150}, "Max": {"X": 504, "Y": 198}} for d in status()["window_draws"]), status()
            pictures += check_window_modal("confirmation")
            key("F7")
            wait_for(lambda s: s.get("step", 0) >= 112_000_000)
            key("F7")
            assert not any(d["logical_rect"] == {"Min": {"X": 200, "Y": 150}, "Max": {"X": 504, "Y": 198}} for d in status()["window_draws"]), status()
            pictures += check_window_modal("after-confirmation")
            key("F7")
    elif is_terrain:
        wait_for(lambda s: s.get("step", 0) >= 740_000_000 and len(s.get("terrain_draws") or []) > 0, 600)
        key("F6")
        key("F7")
        enhanced = capture("enabled")
        modes = list(status()["terrain_modes"])
        draws = list(status()["terrain_draws"])
        sprites = list(status().get("terrain_sprites") or [])
        foreground_reported = status()["terrain_foreground_pixels"]
        key("F6")
        original = capture("original")
        key("F6")
        restored = capture("restored")
        assert enhanced["presented"]["proof"] == original["presented"]["proof"] == restored["presented"]["proof"]
        assert enhanced["presented"]["frame"]["png_sha256"] == restored["presented"]["frame"]["png_sha256"]
        a, b = decoded(Path(enhanced["actual_png"])), decoded(Path(original["actual_png"]))
        allowed = bytearray(1280*800)
        active = []
        foreground = bytearray(1280*800)
        mask_cache = {}
        for draw, mode in zip(draws, modes):
            if mode != "hd":
                continue
            rect = draw["logical_rect"]
            for sprite in sprites:
                sr = sprite["logical_rect"]
                if sprite["draw_order"] <= draw["draw_order"]:
                    continue
                x0, y0 = max(rect["Min"]["X"], sr["Min"]["X"]), max(rect["Min"]["Y"], sr["Min"]["Y"])
                x1, y1 = min(rect["Max"]["X"], sr["Max"]["X"]), min(rect["Max"]["Y"], sr["Max"]["Y"])
                if x0 >= x1 or y0 >= y1:
                    continue
                mask_name = f'{sprite["group"]}-{sprite["index"]:03d}-original.png'
                if mask_name not in mask_cache:
                    mask_cache[mask_name] = decoded(Path("/project/workplace/hd-prototype/catalog/terrain/objects/sources") / mask_name)
                mask = mask_cache[mask_name]
                for y in range(y0, y1):
                    for x in range(x0, x1):
                        i = ((y-sr["Min"]["Y"])*48+x-sr["Min"]["X"])*4
                        if mask[i+3] != 255:
                            continue
                        for dy in range(2):
                            n = (y*2+dy)*1280+x*2
                            foreground[n:n+2] = b'\1\1'
            rect = draw["logical_rect"]
            x0, y0, x1, y1 = rect["Min"]["X"]*2, rect["Min"]["Y"]*2, rect["Max"]["X"]*2, rect["Max"]["Y"]*2
            for y in range(y0, y1):
                allowed[y*1280+x0:y*1280+x1] = bytes([1])*(x1-x0)
            active.append({"index": draw["index"], "rect": [x0//2, y0//2, x1//2, y1//2]})
        changed = 0
        preserved = 0
        for n, eligible in enumerate(allowed):
            if foreground[n]:
                assert a[n*4:n*4+4] == b[n*4:n*4+4], ("original foreground changed", n%1280, n//1280)
                preserved += 1
            if a[n*4:n*4+4] != b[n*4:n*4+4]:
                assert eligible, ("outside validated terrain", n%1280, n//1280)
                changed += 1
        assert (changed > 0) == (args.mode == "terrain")
        assert (len(active) > 0) == (args.mode == "terrain")
        assert preserved == foreground_reported*4
        terrain_checks.append({"step": enhanced["presented"]["step"], "active_rectangles": active,
                               "changed_pixels": changed, "outside_terrain_unchanged": True, "guest_unchanged": True,
                               "foreground_native_pixels": foreground_reported, "foreground_output_pixels": preserved,
                               "original_foreground_unchanged": True, "sprite_draws": sprites})
        pictures += [enhanced, original, restored]
        key("F7")
    elif is_portraits:
        # 105.1M 的普通點擊會開確認框，改在它之前抽樣四張頭像。
        wait_for(lambda s: "76" in (s.get("kao_modes") or {}) and s.get("step", 0) >= 100_000_000)
        key("F6")
        key("F7")
        expected_kao = "hd" if args.mode == "portraits" else "fallback_missing_asset"
        for index in range(76, 80):
            assert status()["kao_modes"][str(index)] == expected_kao, (index, status()["kao_modes"])
        enhanced = capture("kao-76-enabled")
        key("F6")
        original = capture("kao-76-original")
        key("F6")
        restored = capture("kao-76-restored")
        assert enhanced["presented"]["proof"] == original["presented"]["proof"]
        assert enhanced["presented"]["frame"]["png_sha256"] == restored["presented"]["frame"]["png_sha256"]
        a, b = decoded(Path(enhanced["actual_png"])), decoded(Path(original["actual_png"]))
        changed = {index: 0 for index in range(76, 80)}
        for y in range(800):
            for x in range(1280):
                i = (y*1280+x)*4
                head = next((index for index in range(76, 80) if (96+(index-76)*112)*2 <= x < (176+(index-76)*112)*2 and 172 <= y < 332), None)
                if head is None:
                    assert a[i:i+4] == b[i:i+4], ("outside KAO portrait", x, y)
                elif a[i:i+4] != b[i:i+4]:
                    changed[head] += 1
        for index, count in changed.items():
            assert (count > 0) == (expected_kao == "hd"), (index, expected_kao, count)
            kao_checks.append({"index": index, "logical_rect": [96+(index-76)*112, 86, 80, 80], "mode": expected_kao,
                               "step": enhanced["presented"]["step"], "changed_output_pixels": count,
                               "outside_portrait_unchanged": True, "guest_unchanged": True})
        pictures += [enhanced, original, restored]
        key("F7")
        targets = [("A-0", 140_000_000), ("A-1", 170_000_000), ("A-2", 275_000_000),
                   ("A-3", 310_000_000), ("A-4", 365_000_000), ("A-5", 390_000_000),
                   ("A-6", 435_000_000), ("A-7", 492_000_000)]
        if args.include_council:
            targets.append(("GUNGI", 574_000_000))
        labels = {"A-6": [(249, 112, 305, 136)],
                  "A-7": [(102, 54, 146, 82), (160, 35, 202, 61), (110, 112, 171, 140)]}
        for name, step in targets:
            wait_for(lambda s: s.get("eligible") and s.get("portrait_resource") == name + ".OZM" and s.get("step", 0) >= step)
            if not status().get("enabled"):
                key("F6")
                wait_for(lambda s: s.get("enabled"), 10)
            key("F7")
            wait_for(lambda s: s.get("paused"), 10)
            enhanced = capture(name + "-enabled")
            expected = "hd" if name == "A-0" or args.mode == "portraits" else "fallback_missing_asset"
            assert status()["portrait_mode"] == expected, status()
            key("F6")
            original = capture(name + "-original")
            key("F6")
            restored = capture(name + "-restored")
            assert enhanced["presented"]["proof"] == original["presented"]["proof"]
            assert enhanced["presented"]["frame"]["png_sha256"] == restored["presented"]["frame"]["png_sha256"]
            a, b = decoded(Path(enhanced["actual_png"])), decoded(Path(original["actual_png"]))
            changed = 0
            for y in range(500):
                for x in range(1280):
                    i = (y*1280+x)*4
                    portrait = 320 <= x < 960 and 100 <= y < 400
                    if not portrait:
                        assert a[i:i+4] == b[i:i+4], (name, "outside portrait", x, y)
                    elif a[i:i+4] != b[i:i+4]:
                        changed += 1
            assert (changed > 0) == (expected == "hd"), (name, expected, changed)
            preserved = 0
            for x0, y0, x1, y1 in labels.get(name, []):
                for y in range((50+y0)*2, (50+y1)*2):
                    for x in range((160+x0)*2, (160+x1)*2):
                        i = (y*1280+x)*4
                        assert a[i:i+4] == b[i:i+4], (name, "original map label changed", x, y)
                        preserved += 1
            portrait_checks.append({"resource": name + ".OZM", "step": enhanced["presented"]["step"],
                                    "mode": expected, "changed_portrait_pixels": changed,
                                    "preserved_label_pixels": preserved, "outside_portrait_unchanged": True,
                                    "guest_unchanged": True})
            pictures += [enhanced, original, restored]
            key("F7")
    elif args.mode != "mouse":
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
        if args.mode != "missing":
            assert pictures[0]["presented"]["frame"]["mode"] == "hd"
            assert pictures[0]["presented"]["frame"]["png_sha256"] != pictures[1]["presented"]["frame"]["png_sha256"]
        else:
            assert pictures[0]["presented"]["frame"]["mode"] == "fallback_missing_asset"
            assert pictures[0]["presented"]["frame"]["png_sha256"] == pictures[1]["presented"]["frame"]["png_sha256"]
        if is_scroll:
            assert status()["scroll_eligible"], status()
            expected = "fallback_missing_asset" if args.mode == "scroll-missing" else "hd"
            assert status()["scroll_mode"] == expected, status()
            check_scroll_pair(pictures[0], pictures[1])
            key("F7")
            wait_for(lambda s: s.get("step", 0) >= 146_000_000)
            key("F7")
            wait_for(lambda s: s.get("paused"), 10)
            assert status()["scroll_mode"] == expected, status()
            pictures.append(capture("second-enabled"))
            key("F6")
            pictures.append(capture("second-original"))
            key("F6")
            pictures.append(capture("second-restored"))
            assert pictures[3]["presented"]["frame"]["png_sha256"] == pictures[5]["presented"]["frame"]["png_sha256"]
            check_scroll_pair(pictures[3], pictures[4])
            assert scroll_checks[0]["foreground_output_pixels"] != scroll_checks[1]["foreground_output_pixels"], "must exercise different original text pages"
        if is_frame:
            expected = "fallback_missing_asset" if args.mode == "frame-missing" else "hd"
            assert status()["frame_mode"] == expected, status()["frame_mode"]
            assert pictures[0]["presented"]["proof"] == pictures[1]["presented"]["proof"]
            a = decoded(Path(pictures[0]["actual_png"]))
            b = decoded(Path(pictures[1]["actual_png"]))
            changed = preserved = 0
            for y in range(800):
                for x in range(1280):
                    i = (y*1280+x)*4
                    central = 320 <= x < 960 and 100 <= y < 400
                    if y < 500 and not central:
                        changed += a[i:i+4] != b[i:i+4]
                    elif args.mode == "frame-only":
                        assert a[i:i+4] == b[i:i+4], ("frame changed independent layer", x, y)
                        preserved += 1
            assert (changed > 0) == (expected == "hd"), (expected, changed)
            frame_checks.append({"mode": expected, "step": pictures[0]["presented"]["step"],
                                 "changed_frame_pixels": changed, "preserved_other_pixels": preserved,
                                 "guest_unchanged": True})
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
        key("shift+Escape")
    process.wait(timeout=180 if long_path else 60)
    assert process.returncode == 0
    stdout.close()
    stderr.close()
    receipt = json.loads(stdout_path.read_text())
    if args.mode != "mouse":
        assert receipt["successful_steps"] == (750_000_000 if long_path else 150_000_000) and receipt["stop"] == "instruction_budget"
        baseline_path = "/project/workplace/desktop-prototype/headless/startup-probe/latest.json" if long_path else "/project/workplace/hd-prototype/asset-capture/startup-probe/latest.json"
        baseline = json.loads(Path(baseline_path).read_text())
        fields = ["trace_tail", "interrupt_counts", "dpmi_calls", "dpmi_unimplemented", "frame", "cd_pcm",
                  "mounted_disc", "mouse_input_events", "input_boundary_frames"]
        assert all(receipt[k] == baseline[k] for k in fields), [k for k in fields if receipt[k] != baseline[k]]
        expected_ram = "4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9" if long_path else "0d9123e52923ba852b125ea04918e1007138a33e0a8e4d856b078865cb33562f"
        assert hashlib.sha256((root / "startup-memory.bin").read_bytes()).hexdigest() == expected_ram
        toggles = [e for e in receipt["desktop"]["events"] if e["kind"] == "toggle"]
        assert len(toggles) == (3+2*story_cases if is_portraits else 7 if args.mode == "window-modal" else 5 if is_scroll else 3) and all(e["guest_unchanged"] for e in toggles)
        assert any(not e["paused"] for e in toggles)
        if is_portraits:
            assert len(portrait_checks) == story_cases
            assert len(kao_checks) == 4
            loaded = receipt["desktop"]["assets"]["kao"]
            assert len(loaded) == 81 and all(info.get("source_sha256") for info in loaded.values())
            if args.mode == "portraits":
                assert all(info.get("hd_sha256") and not info.get("hd_error") for info in loaded.values())
        elif not is_terrain:
            assert receipt["desktop"]["hd_observation"]["draw_completed"]
        if is_window:
            observation = receipt["desktop"]["hd_observation"]["window"]
            assert observation["pending_draw"] is None and observation["pending_save"] is None and observation["pending_restore"] is None
            kinds = [e["kind"] for e in observation["events"]]
            assert kinds.count("window_save_return") > 0 and kinds.count("window_restore_return") > 0
    else:
        assert receipt["stop"] == "desktop_closed" and receipt["desktop"]["physical_mouse_events"] > 0
        clicks = [e for e in receipt["mouse_input_events"] if e["buttons"] == 1]
        assert clicks and all(abs(e["x"] - 300) <= 1 and abs(e["y"] - 172) <= 1 for e in clicks), clicks
        assert receipt["mouse_input_events"][-1]["buttons"] == 0
        assert pictures[0]["presented"]["frame"]["png_sha256"] != pictures[1]["presented"]["frame"]["png_sha256"]
    result = {"mode": args.mode, "verified": True, "tool": "Xvfb + xdotool + ffmpeg; actual Ebitengine window",
              "receipt_sha256": hashlib.sha256(stdout_path.read_bytes()).hexdigest(), "pictures": pictures,
              "scroll_checks": scroll_checks,
              "portrait_checks": portrait_checks,
              "kao_checks": kao_checks,
              "frame_checks": frame_checks,
              "window_checks": window_checks,
              "terrain_checks": terrain_checks,
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
