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

const threshold = process.env.RESULT_WATCH_THRESHOLD || 10;
const timeout = process.env.RESULT_WATCH_TIMEOUT || 60000;

console.log('executing program:', program);

const handle = client.execute({
  program: program.split('\n').join('').trim(),
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
    .filter(dp => dp.value >= threshold)
    .forEach(dp => {
      console.log("found point that reached the threshold", dp);

      handle.close();
      process.exit(0);
    });
  }
});
