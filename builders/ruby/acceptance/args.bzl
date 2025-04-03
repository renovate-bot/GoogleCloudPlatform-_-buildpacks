"""Module for initializing arguments by Ruby version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}
gae_runtime_versions = {n: v for n, v in gae_runtimes.items()}
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items()}
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)
ruby_gcp_runtime_versions = [key for key in gcp_runtime_versions.keys()]

def rubyargs(runImageTag = "", stack = ""):
    """Create a new key-value map of arguments for Ruby acceptance tests

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for n, v in gae_runtimes.items():
        args[v] = newArgs(n.replace("ruby", ""), runImageTag, stack)
    return args

def newArgs(version, runImageTag, stack):
    return {
        "-run-image-override": runImage(version, runImageTag, stack),
    }

# If an image tag is specified, we get the run image from the 'gae-runtimes-private' repository.
# This can be used in Rapid pipelines and for local testing.
# If no 'runImageTag' is provided, we get the latest runtime image being used in PROD.
def runImage(version, runImageTag, stack):
    # TODO(b/371521232): Newer runtimes do not publish to gcr.
    if version == "34":
        return "us-docker.pkg.dev/serverless-runtimes/google-22-full/runtimes/ruby34:latest"
    if stack != "":
        if runImageTag != "":
            return "us-docker.pkg.dev/gae-runtimes-private/" + stack + "/runtimes/ruby" + version + ":" + runImageTag
        else:
            return "gcr.io/gae-runtimes/buildpacks/ruby" + version + "/run"

    if runImageTag != "":
        return "us-docker.pkg.dev/gae-runtimes-private/gcp/buildpacks/ruby" + version + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/ruby" + version + "/run"
