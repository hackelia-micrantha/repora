#!/usr/bin/env python3
import os
import sys

CHUNK_BYTES = 64 * 1024


def main(argv):
    if len(argv) != 3:
        print(f"usage: {argv[0]} <output> <max-bytes>", file=sys.stderr)
        return 2

    destination = argv[1]
    try:
        maximum = int(argv[2])
    except ValueError:
        print("max-bytes must be an integer", file=sys.stderr)
        return 2
    if maximum <= 0:
        print("max-bytes must be greater than zero", file=sys.stderr)
        return 2

    try:
        fd = os.open(destination, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    except OSError as exc:
        print(f"cannot create capture {destination!r}: {exc}", file=sys.stderr)
        return 1

    observed = 0
    retained = 0
    try:
        with os.fdopen(fd, "wb") as output:
            while True:
                block = sys.stdin.buffer.read(CHUNK_BYTES)
                if not block:
                    break
                observed += len(block)
                room = maximum - retained
                if room > 0:
                    piece = block[:room]
                    output.write(piece)
                    retained += len(piece)
    except OSError as exc:
        print(f"capture write failed for {destination!r}: {exc}", file=sys.stderr)
        return 1

    if observed > maximum:
        print(
            f"capture exceeded {maximum} bytes (observed {observed}); "
            "retained bytes must not be used as evidence input",
            file=sys.stderr,
        )
        return 3
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
