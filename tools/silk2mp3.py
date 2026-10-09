#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
silk2mp3.py —— 把微信/QQ 的 SILK v3 语音批量转成 mp3。

依赖：
  - pilk   : 纯 Python 的 SILK 解码器（pip install pilk）
  - ffmpeg : 把解码出的 PCM 转 mp3（需在 PATH，或用 --ffmpeg 指定）

用法：
  python silk2mp3.py <目录或文件> [--ffmpeg ffmpeg.exe] [--rate 24000] [--keep-silk]

行为：
  - 递归遍历目录下所有 *.silk，逐个转成同名 *.mp3（放在原目录）。
  - 默认转换成功后删除 .silk（--keep-silk 可保留）。
  - 已存在同名 .mp3 的跳过。
  - 打印进度与统计。
"""
import argparse
import os
import subprocess
import sys
import tempfile

try:
    import pilk
except ImportError:
    sys.stderr.write("[!] 缺少 pilk，请先执行: pip install pilk\n")
    sys.exit(2)


def find_ffmpeg(explicit: str) -> str:
    if explicit:
        return explicit
    # 常见位置兜底
    for cand in ("ffmpeg", r"D:\down\ffmpeg\bin\ffmpeg.exe"):
        try:
            subprocess.run([cand, "-version"], stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL, check=True)
            return cand
        except Exception:
            continue
    return ""


def convert_one(silk_path: str, ffmpeg: str, rate: int, keep_silk: bool) -> bool:
    mp3_path = os.path.splitext(silk_path)[0] + ".mp3"
    if os.path.exists(mp3_path) and os.path.getsize(mp3_path) > 0:
        return True
    fd, pcm_path = tempfile.mkstemp(suffix=".pcm")
    os.close(fd)
    try:
        pilk.decode(silk_path, pcm_path)
        if not os.path.exists(pcm_path) or os.path.getsize(pcm_path) == 0:
            return False
        cmd = [ffmpeg, "-y", "-loglevel", "error",
               "-f", "s16le", "-ar", str(rate), "-ac", "1",
               "-i", pcm_path,
               "-codec:a", "libmp3lame", "-b:a", "64k",
               mp3_path]
        r = subprocess.run(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        if r.returncode != 0 or not os.path.exists(mp3_path):
            return False
        if not keep_silk:
            try:
                os.remove(silk_path)
            except OSError:
                pass
        return True
    except Exception as e:
        sys.stderr.write(f"    [!] {os.path.basename(silk_path)}: {e}\n")
        return False
    finally:
        try:
            os.remove(pcm_path)
        except OSError:
            pass


def iter_silk(root: str):
    if os.path.isfile(root):
        if root.lower().endswith(".silk"):
            yield root
        return
    for dirpath, _dirs, files in os.walk(root):
        for f in files:
            if f.lower().endswith(".silk"):
                yield os.path.join(dirpath, f)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("path", help="目录或单个 .silk 文件")
    ap.add_argument("--ffmpeg", default="", help="ffmpeg 可执行文件路径")
    ap.add_argument("--rate", type=int, default=24000, help="采样率（默认 24000）")
    ap.add_argument("--keep-silk", action="store_true", help="保留原始 .silk")
    args = ap.parse_args()

    ffmpeg = find_ffmpeg(args.ffmpeg)
    if not ffmpeg:
        sys.stderr.write("[!] 找不到 ffmpeg，请用 --ffmpeg 指定路径\n")
        sys.exit(2)

    files = list(iter_silk(args.path))
    if not files:
        print("[i] 没有找到 .silk 文件")
        return

    ok = fail = 0
    for i, p in enumerate(files, 1):
        if convert_one(p, ffmpeg, args.rate, args.keep_silk):
            ok += 1
        else:
            fail += 1
        if i % 50 == 0 or i == len(files):
            print(f"  进度 {i}/{len(files)}  成功 {ok}  失败 {fail}")
    print(f"[OK] 语音转码完成：成功 {ok}，失败 {fail}，共 {len(files)}")


if __name__ == "__main__":
    main()
