import unittest

from validate_release import validate


class ReleaseValidationTests(unittest.TestCase):
    repository = "maintainer/library"
    module = "github.com/" + repository

    def test_compatible_versions(self):
        for tag, suffix in [
            ("v0.1.0", ""), ("v1.0.0", ""), ("v2.0.0", "/v2"),
            ("v10.2.3", "/v10"), ("v1.0.0-rc.1", ""),
            ("v1.0.0-0.alpha-1", ""),
        ]:
            with self.subTest(tag=tag):
                validate(tag, self.module + suffix, self.repository)

    def test_malformed_tags(self):
        for tag in [
            "1.0.0", "v1", "v1.0", "v01.0.0", "v1.00.0", "v1.0.00",
            "v1.0.0-", "v1.0.0-rc..1", "v1.0.0-01", "v1.0.0-rc.01",
            "v1.0.0+build", "v1.0.0\n", "v1.0.0-rc/1", "v1.0.0-é",
        ]:
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                validate(tag, self.module, self.repository)

    def test_module_mismatches(self):
        for tag, module in [
            ("v2.0.0", self.module), ("v1.0.0", self.module + "/v2"),
            ("v0.1.0", self.module + "/v1"), ("v2.0.0", self.module + "/v3"),
            ("v2.0.0", self.module + "/v02"), ("v0.1.0", "local-module"),
            ("v0.1.0", "github.com/someone/another-library"),
        ]:
            with self.subTest(tag=tag, module=module), self.assertRaises(ValueError):
                validate(tag, module, self.repository)


if __name__ == "__main__":
    unittest.main()
