# fs-dedup

Out-of-band file system deduplication utility.

Make sure there is no unexpected change will appear on the locations or something wrong may happen.

## How it works?

Step 1: Seek files with the same size.

Step 2: Seek files with the same checksum (SHA256).

Step 3: Ref-link via `unix.IoctlFileClone`.

## Usage

```
Usage: fs-dedup [OPTIONS] (FILE)...
  -R    Recursively scan directories.
  -min-size int
        Minimum size. (default 1048576)
```
