local gocdtasks = import 'github.com/getsentry/gocd-jsonnet/libs/gocd-tasks.libsonnet';

local deploy_canary_stage(region) =
  if region == 'us' then
    [
      {
        'deploy-canary': {
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
    ]
  else
    [];

function(region) {
  environment_variables: {
    GITHUB_TOKEN: '{{SECRET:[devinfra-github][token]}}',
    SENTRY_REGION: region,
    SKIP_CANARY_CHECKS: false,
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
            ],
          },
        },
      },
    },
  ] + deploy_canary_stage(region) + [
    {
      'deploy-primary': {
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
