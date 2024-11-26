"""
Proxy funtioanl testing for Axlearn(tensforflow) libraries to interface GCS

Setup:
  Set the following the enviroment variables
   -- PROXY_FUNC_TEST_KMS_KEY: KMS key used by proxy. Required
   -- PROXY_FUNC_TEST_BUCKET: GCS bucket for testing. Optional
   -- https_proxy: Point to the proxy. Required
                   ie. https_proxy=https://localhost:8080
   -- REQUESTS_CA_BUNDLE: Point to the proxy ca cert. Requred

Usage:
  >>> pytest -v -s --log-cli-level=INFO test_axlearn_tf.py

"""
import os
import pytest
import logging
import test_util
import uuid


LOG_LEVEL_STR = os.environ.get("PROXY_FUNC_TEST_LOG_LEVEL", "INFO")
log_level = getattr(logging, LOG_LEVEL_STR.upper(), logging.INFO)
logging.basicConfig(level=logging.INFO,
                    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')

logger = logging.getLogger(__name__)

GCP_KMS_KEY = os.environ.get(
    "PROXY_FUNC_TEST_KMS_KEY",
    "gcp-kms://projects/your-project/locations/global/keyRings/your-key-ring/cryptoKeys/your-key",
)
TEST_BUCKET = os.environ.get(
    "PROXY_FUNC_TEST_BUCKET",
    "gcs-proxy-func-test",
)
OBJECT_NAME_PREFIX = "func-test"
OBJECT_CONTENT = "testing object content"

@pytest.fixture(scope="module")
def setup_data():
    """Fixture to set up any necessary data or resources."""
    rand_id = uuid.uuid4()
    object_path_byte = (
        f"gs://{TEST_BUCKET}/{rand_id}/{OBJECT_NAME_PREFIX}-{rand_id}").encode("utf-8")
    env_aead = test_util.get_aead(GCP_KMS_KEY)
    orginal_object_byte = OBJECT_CONTENT.encode("utf-8")
    encrypted_object_byte = env_aead.encrypt(
        orginal_object_byte, object_path_byte)
    logger.info(f"test run uuid: {rand_id}")
    logger.info(f"object_path: {object_path_byte}")
    return {
        "original_object_byte": orginal_object_byte,
        "encrypted_object_byte": encrypted_object_byte,
        "env_aead": env_aead,
        "object_path_byte": object_path_byte
    }


def test_tink_decrypt(setup_data):
    exepcted = setup_data["original_object_byte"]
    env_aead = setup_data["env_aead"]
    actual = env_aead.decrypt(
        setup_data["encrypted_object_byte"], setup_data["object_path_byte"])
    assert actual == exepcted


def test_axlearn_fileio_copy(setup_data):
    """Test case for axlearn file_io.copy()"""
    assert True

def test_tf_io_gfile_write(setup_data):
    """Test case for tf.io.gfile.GFile.write()"""
    assert True


def test_tf_io_gfile_read(setup_data):
    """Test case for tf.io.gfile.GFile.read()"""
    assert True

def test_tf_data_read(setup_data):
    """Test case for tf.data.TFRecordDataset which is used by axlearn input_tf_data.tfrecrod_dataset"""
    assert True

def test_tf_tensorstore_read(setup_data):
    """Test case for tensortore - read from GCS"""
    assert True

def test_tf_tensorstore_write(setup_data):
    """Test case for tensortore - write to GCS. i.e orbax checkpoint."""
    assert True

def test_tf_summary_write(setup_data):
    """Test case for tf.summary - write to GCS. i.e tf native checkpoint."""
    assert True

def test_tfds_read(setup_data):
    """Test case for tensorflow-dataset. Load public curated data from GCS"""
    assert True

# Add more test functions as needed

if __name__ == "__main__":
    pytest.main()