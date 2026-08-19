import {cpSync,mkdirSync} from 'node:fs';mkdirSync('dist',{recursive:true});for(const f of ['index.html','app.js','styles.css'])cpSync('src/'+f,'dist/'+f);
