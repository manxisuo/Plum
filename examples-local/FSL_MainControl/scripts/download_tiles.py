#!/usr/bin/env python3
"""
Download a small set of OpenStreetMap tiles for offline use.

This script should be executed在联网环境中，下载完成后将生成的 tiles 目录
拷贝到本项目的 examples-local/FSL_MainControl/static/tiles 下即可在离线环境使用。

默认下载以 (30.673916, 122.973926) 为中心，半径约 6km 的区域。
可通过命令行参数调整：

    python3 download_tiles.py --min-zoom 11 --max-zoom 15 \
        --lat 30.673916 --lon 122.973926 --radius 0.05
"""

import argparse
import math
import os
import pathlib
import queue
import sys
import threading
import time
from typing import Tuple

import requests

TILE_SERVER = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
REQUEST_TIMEOUT = 10
THREAD_COUNT = 6


def deg2num(lat_deg: float, lon_deg: float, zoom: int) -> Tuple[int, int]:
    lat_rad = math.radians(lat_deg)
    n = 2.0 ** zoom
    xtile = int((lon_deg + 180.0) / 360.0 * n)
    ytile = int(
        (1.0 - math.log(math.tan(lat_rad) + (1 / math.cos(lat_rad))) / math.pi) / 2.0 * n
    )
    return xtile, ytile


def download_tile(z: int, x: int, y: int, dest_dir: pathlib.Path):
    url = TILE_SERVER.format(z=z, x=x, y=y)
    dest_path = dest_dir / str(z) / str(x)
    dest_path.mkdir(parents=True, exist_ok=True)
    file_path = dest_path / f"{y}.png"
    if file_path.exists():
        return
    try:
        resp = requests.get(url, timeout=REQUEST_TIMEOUT, headers={"User-Agent": "FSL-Tile-Downloader"})
        resp.raise_for_status()
        file_path.write_bytes(resp.content)
    except Exception as exc:
        print(f"[WARN] Failed to fetch tile z={z} x={x} y={y}: {exc}")


def worker(task_queue: "queue.Queue[Tuple[int, int, int]]", dest_dir: pathlib.Path, total: int):
    processed = 0
    while True:
        try:
            zxy = task_queue.get_nowait()
        except queue.Empty:
            break
        z, x, y = zxy
        download_tile(z, x, y, dest_dir)
        processed += 1
        if processed % 50 == 0:
            print(f"[INFO] thread {threading.current_thread().name} processed {processed} tiles")
        task_queue.task_done()


def main():
    parser = argparse.ArgumentParser(description="Download offline map tiles for FSL demo")
    parser.add_argument("--lat", type=float, default=30.664554, help="中心点纬度")
    parser.add_argument("--lon", type=float, default=122.510268, help="中心点经度")
    parser.add_argument(
        "--radius",
        type=float,
        default=0.05,
        help="经纬度范围（单位：度），0.05 大约对应 ±5~6km",
    )
    parser.add_argument("--min-zoom", type=int, default=11, help="最小缩放级别")
    parser.add_argument("--max-zoom", type=int, default=15, help="最大缩放级别")
    parser.add_argument(
        "--output",
        type=pathlib.Path,
        default=pathlib.Path(__file__).resolve().parent.parent / "static" / "tiles",
        help="输出目录（默认写到 static/tiles）",
    )
    args = parser.parse_args()

    if args.max_zoom < args.min_zoom:
        print("max_zoom 必须 >= min_zoom", file=sys.stderr)
        return 1

    dest_dir = args.output
    dest_dir.mkdir(parents=True, exist_ok=True)

    task_queue: "queue.Queue[Tuple[int, int, int]]" = queue.Queue()

    total_tasks = 0
    for zoom in range(args.min_zoom, args.max_zoom + 1):
        min_lat = args.lat - args.radius
        max_lat = args.lat + args.radius
        min_lon = args.lon - args.radius
        max_lon = args.lon + args.radius

        x_start, y_start = deg2num(max_lat, min_lon, zoom)
        x_end, y_end = deg2num(min_lat, max_lon, zoom)

        x_min, x_max = sorted((x_start, x_end))
        y_min, y_max = sorted((y_start, y_end))

        for x in range(x_min, x_max + 1):
            for y in range(y_min, y_max + 1):
                task_queue.put((zoom, x, y))
                total_tasks += 1

    print(f"[INFO] 准备下载 {total_tasks} 张瓦片到 {dest_dir}")

    threads = []
    for _ in range(min(THREAD_COUNT, total_tasks)):
        t = threading.Thread(target=worker, args=(task_queue, dest_dir, total_tasks), daemon=True)
        threads.append(t)
        t.start()

    start_time = time.time()
    for t in threads:
        t.join()

    elapsed = time.time() - start_time
    print(f"[INFO] 完成，耗时 {elapsed:.1f}s，文件保存在 {dest_dir}")
    print("[INFO] 下载数据仅供调试使用，请遵守 OpenStreetMap 使用条款。")
    return 0


if __name__ == "__main__":
    sys.exit(main())

