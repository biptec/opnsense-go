import importlib.util
from pathlib import Path
import unittest


MODULE_PATH = Path(__file__).with_name("install-bind-plugin.py")
SPEC = importlib.util.spec_from_file_location("install_bind_plugin", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class BuildInstallScriptTest(unittest.TestCase):
    def test_uses_version_independent_package_and_commit_verification(self):
        commit = "fc897dbb8fbcc1d33fcb87239297d2960a893893"
        script = MODULE.build_install_script(commit)

        self.assertIn(commit, script)
        self.assertIn("pkg install -y bind920", script)
        self.assertLess(script.index("pkg install -y bind920"), script.index("make -C /usr/plugins/dns/bind upgrade"))
        self.assertIn("pkg info -e 'os-bind-*'", script)
        self.assertIn("product_hash", script)
        self.assertIn('case "' + commit + '" in', script)
        self.assertNotIn("os-bind-1.35", script)
        self.assertNotIn("os-bind-1.36", script)


if __name__ == "__main__":
    unittest.main()
