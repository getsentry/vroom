from devenv import constants
from devenv.lib import brew, config, proc, uv


def main(context: dict[str, str]) -> int:
    reporoot = context["reporoot"]
    cfg = config.get_repo(reporoot)

    brew.install()

    proc.run(
        (f"{constants.homebrew_bin}/brew", "bundle"),
        cwd=reporoot,
    )

    print("installing goimports...")
    proc.run(("go", "install", "golang.org/x/tools/cmd/goimports@v0.36.0"))

    uv.install(
        cfg["uv"]["version"],
        cfg["uv"][constants.SYSTEM_MACHINE],
        cfg["uv"][f"{constants.SYSTEM_MACHINE}_sha256"],
        reporoot,
    )

    print("syncing .venv ...")
    proc.run(
        (f"{reporoot}/.devenv/bin/uv", "sync", "--frozen", "--quiet", "--active"),
    )

    print("installing pre-commit hooks ...")
    proc.run((f"{reporoot}/.venv/bin/pre-commit", "install", "--install-hooks"))

    return 0
