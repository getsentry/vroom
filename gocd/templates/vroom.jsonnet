local vroom = import './pipelines/vroom.libsonnet';
local pipedream = import 'github.com/getsentry/gocd-jsonnet/libs/pipedream.libsonnet';

local pipedream_config = {
  name: 'vroom',
  materials: {
    vroom_repo: {
      git: 'git@github.com:getsentry/vroom.git',
      shallow_clone: true,
      branch: 'main',
      destination: 'vroom',
    },
  },
  rollback: {
    material_name: 'vroom_repo',
    stage: 'deploy-primary',
    elastic_profile_id: 'vroom',
  },
};

pipedream.render(pipedream_config, vroom)
