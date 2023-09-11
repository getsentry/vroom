local gocdtasks = import 'github.com/getsentry/gocd-jsonnet/libs/gocd-tasks.libsonnet';

function(region) {
  environment_variables: {
    // SENTRY_REGION is used by the dev-infra scripts to connect to GKE
    SENTRY_REGION: region,
  },
  materials: {
    vroom_repo: {
      git: 'git@github.com:getsentry/vroom.git',
      shallow_clone: true,
      branch: 'main',
      destination: 'vroom',
    },
  },
  lock_behavior: 'unlockWhenFinished',
  stages: [
    {
      deploy_primary: {
        approval: {
          type: 'manual',
        },
        fetch_materials: true,
        jobs: {
          deploy: {
            timeout: 1200,
            elastic_profile_id: 'example',
            environment_variables: {
              LABEL_SELECTOR: 'service=example',
            },
            tasks: [
              gocdtasks.noop,
            ],
          },
        },
      },
    },
  ],
}
