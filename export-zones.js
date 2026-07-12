import { zones } from './dashboard/src/utils/zones.js';
import fs from 'fs';
fs.writeFileSync('engine/zones.json', JSON.stringify(zones));
console.log('Zones exported to engine/zones.json');
