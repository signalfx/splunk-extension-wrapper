const signalfx = require('signalfx');

const program = `
  data('lambda.function.invocation', 
    filter=filter('aws_function_name', '${process.env.FUNCTION_NAME}'), 
    rollup='sum',
    extrapolation='zero')
  .sum()
  .sum(over='5m')
  .publish()
`.split('\n').join('').trim();

const options = {
  'signalflowEndpoint': `wss://stream.${process.env.FUNCTION_REALM}.signalfx.com`,
  'webSocketErrorCallback': evt => {
    console.error('web socket error', evt);
    process.exit(1);
  }
};

const client = new signalfx.SignalFlow(process.env.FUNCTION_TOKEN, options);

const expectedInvocationCount = process.env.EXPECTED_INVOCATION_COUNT || 10;
const timeout = process.env.TEST_VERIFICATION_TIMEOUT || 60000;

console.log('executing program:', program);

const handle = client.execute({
  program: program,
  stop: Date.now() + timeout,
  resolution: 1000,
  immediate: false
});

handle.stream((err, data) => {
  if (err) {
    console.error('encountered an error', err);

    process.exit(1);
  }

  if (data.type === 'control-message' && data.event === 'END_OF_CHANNEL') {
    console.error('channel closed before reaching the threshold');

    handle.close();
    process.exit(1);
  }

  console.log(data);

  if (data.type === 'data') {
    data.data
    .filter(dp => dp.value >= expectedInvocationCount)
    .forEach(dp => {
      console.log("the threshold has been reached", dp);

      handle.close();
      process.exit(0);
    });
  }
});
