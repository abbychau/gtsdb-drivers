// Example: Node.js GTSDB driver usage.
//
//   node example.js [host:port]

const { GTSDBClient } = require('./gtsdb.js');

async function main() {
  const addr = process.argv[2] || 'localhost:5555';
  const [host, port] = addr.split(':');

  console.log('Connecting to', addr, '...');
  const client = new GTSDBClient(host, parseInt(port) || 5555);
  await client.connect();
  console.log('✓ Connected');

  // Auth (skip if no-auth mode)
  try { await client.auth(''); } catch {}

  // Write
  process.stdout.write('Write... ');
  await client.write('test.sensor', 42.5);
  console.log('✓');

  // Write with timestamp
  process.stdout.write('WriteAt... ');
  await client.write('test.sensor', 99.9, 1717965210);
  console.log('✓');

  // Read JSON
  process.stdout.write('ReadLast JSON... ');
  let pts = await client.readLast('test.sensor', 10);
  console.log(`✓ (${pts.length} points)`);
  if (pts.length > 0) {
    const last = pts[pts.length - 1];
    console.log(`  latest: ts=${last.timestamp} val=${last.value}`);
  }

  // Read binary 🚀
  process.stdout.write('ReadLast Binary... ');
  pts = await client.readBinary('test.sensor', 10);
  console.log(`✓ (${pts.length} points)`);
  if (pts.length > 0) {
    const last = pts[pts.length - 1];
    console.log(`  latest: ts=${last.timestamp} val=${last.value}`);
  }

  // Batch write
  process.stdout.write('BatchWrite... ');
  await client.batchWrite([
    { key: 'test.batch1', value: 1.0 },
    { key: 'test.batch2', value: 2.0 },
    { key: 'test.batch3', value: 3.0 },
  ]);
  console.log('✓');

  // IDs
  process.stdout.write('IDs... ');
  const ids = await client.ids();
  console.log(`✓ (${ids.length} keys: ${ids.slice(0, 3)})`);

  await client.close();
  console.log('\nAll tests passed!');
}

main().catch(err => { console.error('FAIL:', err.message); process.exit(1); });
