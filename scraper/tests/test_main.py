import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))


def test_ca_bundle_includes_certifi_and_extras():
    from main import ensure_ca_bundle
    import certifi

    bundle_path = ensure_ca_bundle()
    bundle = open(bundle_path).read()
    extra_path = os.path.join(os.path.dirname(__file__), "..", "certs", "letsencrypt-yr.pem")
    assert open(certifi.where()).read() in bundle
    assert open(extra_path).read() in bundle


def test_main_no_args(capsys):
    sys.argv = ["main.py"]
    try:
        from main import main
        main()
    except SystemExit as e:
        assert e.code == 1


def test_main_unknown_command(capsys):
    sys.argv = ["main.py", "unknown"]
    try:
        from main import main
        main()
    except SystemExit as e:
        assert e.code == 1


def test_main_scrape_missing_url(capsys):
    sys.argv = ["main.py", "scrape"]
    try:
        from main import main
        main()
    except SystemExit as e:
        assert e.code == 1
