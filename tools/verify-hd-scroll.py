"""沿用有界桌面驗證器，檢查兩頁劇情捲軸與分項缺圖備援。"""
import argparse
from pathlib import Path
import runpy
import sys

parser = argparse.ArgumentParser()
parser.add_argument("--mode", choices=["hd", "missing", "only"], default="hd")
mode = parser.parse_args().mode
target = Path(__file__).with_name("verify-desktop.py")
sys.argv = [str(target), "--mode", {"hd": "scroll", "missing": "scroll-missing", "only": "scroll-only"}[mode]]
runpy.run_path(str(target), run_name="__main__")
