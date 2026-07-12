from upanime_worker.callbacks import build_failure_callback, build_success_callback


def test_build_success_callback_sets_expected_fields():
    payload = build_success_callback(42, "animes/naruto/ep_1_upscaled.mp4")

    assert payload.job_id == 42
    assert payload.status == "completed"
    assert payload.result_storage_key == "animes/naruto/ep_1_upscaled.mp4"
    assert payload.file_name == "ep_1_upscaled.mp4"


def test_build_failure_callback_sets_expected_fields():
    payload = build_failure_callback(42, "gpu offline", "animes/naruto/ep_1_upscaled.mp4")

    assert payload.job_id == 42
    assert payload.status == "failed"
    assert payload.error == "gpu offline"
    assert payload.result_storage_key == "animes/naruto/ep_1_upscaled.mp4"
    assert payload.file_name == "ep_1_upscaled.mp4"
