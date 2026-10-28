"""Validate root GitHub module tags without fetching or changing repository state."""

import re
import sys


TAG = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z.-]+))?", re.ASCII)


def validate(tag, module_path, repository):
    match = TAG.fullmatch(tag)
    if match is None:
        raise ValueError("expected vMAJOR.MINOR.PATCH with an optional SemVer prerelease; build metadata is not supported")
    prerelease = match.group(4)
    if prerelease is not None:
        for identifier in prerelease.split("."):
            if not identifier or (identifier.isdecimal() and len(identifier) > 1 and identifier.startswith("0")):
                raise ValueError("invalid SemVer prerelease identifier")

    major = int(match.group(1))
    expected = "github.com/" + repository
    if major >= 2:
        expected += "/v" + str(major)
    if module_path != expected:
        raise ValueError(f"{tag} requires module path {expected}; found {module_path}")


if __name__ == "__main__":
    if len(sys.argv) != 4:
        sys.exit("usage: validate_release.py TAG MODULE_PATH OWNER/REPOSITORY")
    try:
        validate(*sys.argv[1:])
    except ValueError as error:
        sys.exit(str(error))
    print("Release tag and module path are compatible.")
