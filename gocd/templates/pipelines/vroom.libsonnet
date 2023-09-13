local gocdtasks = import 'github.com/getsentry/gocd-jsonnet/libs/gocd-tasks.libsonnet';

function(region) {
  environment_variables: {
    // SENTRY_REGION is used by the dev-infra scripts to connect to GKE
    SENTRY_REGION: region,
    GITHUB_TOKEN: '{{SECRET:[devinfra-github][token]}}',
    GCP_PROJECT: 'internal-sentry',
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
      checks: {
        fetch_materials: true,
        jobs: {
          deploy: {
            timeout: 600,
            elastic_profile_id: 'vroom',
            tasks: [
              gocdtasks.script(importstr '../bash/check-github.sh'),
              gocdtasks.script(importstr '../bash/check-cloudbuild.sh'),
            ],
          },
        },
      },
    },
    {
      deploy_canary: {
        fetch_materials: true,
        jobs: {
          deploy: {
            timeout: 600,
            elastic_profile_id: 'vroom',
            environment_variables: {
              LABEL_SELECTOR: 'service=vroom,environment=production,env=canary',
              WAIT_MINUTES: '5',
            },
            tasks: [
              gocdtasks.script(importstr '../bash/deploy.sh'),
              gocdtasks.script(importstr '../bash/wait-canary.sh'),
            ],
          },
        },
      },
    },
    {
      deploy_primary: {
        fetch_materials: true,
        jobs: {
          deploy: {
            timeout: 600,
            elastic_profile_id: 'vroom',
            environment_variables: {
              LABEL_SELECTOR: 'service=vroom,environment=production',
            },
            tasks: [
              gocdtasks.script(importstr '../bash/deploy.sh'),
            ],
          },
        },
      },
    },
  ],
}
