from pathlib import Path


def test_docker_env_example_contains_required_worker_variables():
    env_file = Path(__file__).resolve().parents[2] / "docker.env.example"
    contents = env_file.read_text()

    assert "WORKER_MODEL_PATH=/data/models/realesr-animevideov3.pth" in contents
    assert "R2_ACCOUNT_ID=" in contents
    assert "R2_BUCKET_NAME=" in contents
    assert "WORKER_HOST" not in contents
    assert "WORKER_PORT" not in contents
