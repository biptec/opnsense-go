import importlib.util
import pathlib
import tempfile
import unittest
from unittest import mock

SCRIPT = pathlib.Path(__file__).with_name("create-apikey.py")
spec = importlib.util.spec_from_file_location("create_apikey", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class CreateApiKeyTests(unittest.TestCase):
    def test_parse_credentials(self):
        key, secret = module.parse_credentials("key=abc\nsecret=xyz\n")
        self.assertEqual((key, secret), ("abc", "xyz"))

    def test_parse_credentials_rejects_incomplete_output(self):
        with self.assertRaises(RuntimeError):
            module.parse_credentials("key=abc\n")

    def test_wait_guest_agent_retries(self):
        with mock.patch.object(module, "send_qemu_command", side_effect=[OSError("not ready"), {}]) as send, \
             mock.patch.object(module.time, "sleep"):
            module.wait_guest_agent(timeout=5)
        self.assertEqual(send.call_count, 2)

    def test_write_outputs_only_writes_output_records(self):
        with tempfile.NamedTemporaryFile() as output, mock.patch.dict(module.os.environ, {"GITHUB_OUTPUT": output.name}):
            module.write_outputs("abc", "xyz")
            output.seek(0)
            self.assertEqual(output.read().decode(), "key=abc\nsecret=xyz\n")


if __name__ == "__main__":
    unittest.main()
