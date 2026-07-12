from pathlib import Path


def test_worker_dockerfile_contains_runtime_dependencies_and_entrypoint():
    dockerfile = Path(__file__).resolve().parents[2] / "Dockerfile"
    contents = dockerfile.read_text()

    assert "FROM runpod/pytorch:" in contents
    assert "ffmpeg" in contents
    assert "COPY src ./src" in contents
    assert 'ENTRYPOINT ["python", "-m", "upanime_worker.handler"]' in contents
    assert "EXPOSE" not in contents
