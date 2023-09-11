// Import a jsonnet function that returns a the GoCD pipeline object
local vroom = import './pipelines/vroom.libsonnet';
// Import the pipedream library which is imported by jsonnet-bundler to ./vendor
local pipedream = import 'github.com/getsentry/gocd-jsonnet/libs/pipedream.libsonnet';

// Pipedream can be configured using this object, you can learn more about the
// configuration options here: https://github.com/getsentry/gocd-jsonnet#readme
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
  exclude_regions: [
    'customer-1',
    'customer-2',
    'customer-3',
    'customer-4',
    'customer-5',
    'customer-6',
  ],
};

// Calling pipedream.render will return an object that contains all the pipelines
// needed for a full pipedream deployment.
// pipedream.render will call the pipeline function we import at the top
// of this file, once for each region.
pipedream.render(pipedream_config, vroom)
