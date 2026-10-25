import { spawn, execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { lstat, mkdir, readdir, readFile, realpath, writeFile } from 'node:fs/promises';
import { basename, dirname, isAbsolute, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

// Manual model evaluation: eight provider-backed sessions, never run by CI.
const options=new Map();const arguments_=process.argv.slice(2);
for(let index=0;index<arguments_.length;index+=2){
 if(!['--destination','--ctx'].includes(arguments_[index]) || !arguments_[index+1] || options.has(arguments_[index])) throw Error('Usage: --destination /absolute/new/path --ctx /absolute/ctx');
 options.set(arguments_[index],arguments_[index+1]);
}
if(!isAbsolute(options.get('--destination')||'') || !isAbsolute(options.get('--ctx')||''))throw Error('Absolute destination and CLI paths are required');
const requested=resolve(options.get('--destination'));
const root=join(await realpath(dirname(requested)),basename(requested));
try{await lstat(root);throw Error('Refusing existing evaluation destination: '+root);}catch(error){if(error.code!=='ENOENT')throw error;}
const cli=await realpath(options.get('--ctx'));
execFileSync(cli,['--version'],{stdio:'pipe'});
const checkout=resolve(dirname(fileURLToPath(import.meta.url)),'..');
const scopeGuard=join(checkout,'evals/ctx-skill/fixtures/retrieval/scope-guard.mjs');
await mkdir(root);
const tasks={
 focused:'What precisely does ctx mean by hydration? In a fresh clone of an existing team scaffold, what can init create, what does it preserve, and can it change installed add-ons? Explain the important refusal boundaries and cite source.',
 cross:'We have a flat layout-v2 team scaffold. A proposed plan is to run ctx update to get folder-first files and the latest default add-ons, then treat a verified category index as proof that every linked topic is accurate. Is that correct? Explain a safe in-place way to add contracts, what doctor and status can establish, and how index and topic readiness relate. Cite source.'
};
const models=[{name:'glm',provider:'ollama',model:'glm-5.3:cloud',thinking:null},{name:'opus',provider:'anthropic',model:'claude-opus-4-8',thinking:'high'}];
const hash=data=>createHash('sha256').update(data).digest('hex');
const git=(repo,...args)=>execFileSync('git',['-C',repo,...args],{encoding:'utf8'}).trim();
async function manifest(repo){
 const entries={};
 async function visit(dir='') {for(const name of (await readdir(join(repo,dir))).sort()) {
  if(!dir && name==='.git') continue;
  const path=dir?dir+'/'+name:name;const info=await lstat(join(repo,path));
  entries[path]={type:info.isDirectory()?'directory':info.isFile()?'file':'other',mode:info.mode & 0o777};
  if(info.isDirectory()) await visit(path); else if(info.isFile()) entries[path].hash=hash(await readFile(join(repo,path)));
 }}
 await visit();
 return {entries,head:git(repo,'rev-parse','HEAD'),staged:git(repo,'diff','--cached'),index:hash(await readFile(join(repo,'.git/index')))};
}
const schedule=[];let arm=0;
for(const task of ['focused','cross']) for(const model of models) for(const layout of (arm++%2 ? ['folder','flat']:['flat','folder'])) schedule.push({task,model,layout});
const prepared=[];
for(const [index,trial] of schedule.entries()){
 const repo=join(root,'workspace-'+String(index+1).padStart(2,'0'));
 const input=JSON.parse(execFileSync(process.execPath,[join(checkout,'tools/prepare-retrieval-comparison.mjs'),'--destination',repo,'--layout',trial.layout,'--ctx',cli],{encoding:'utf8'}));
 prepared.push({...trial,repo,input});
}
for(const trial of prepared){
 if(trial.input.semanticHash!==prepared[0].input.semanticHash || trial.input.skillHash!==prepared[0].input.skillHash || JSON.stringify(trial.input.sourceHashes)!==JSON.stringify(prepared[0].input.sourceHashes)) throw Error('Comparison inputs differ');
}
await writeFile(join(root,'inputs.json'),JSON.stringify(prepared,null,2));
console.log(JSON.stringify({stage:'prepared',trials:prepared.length,semanticHash:prepared[0].input.semanticHash}));
const records=[];
for(const trial of prepared){
 const before=await manifest(trial.repo);
 const prompt=`Work only in ${trial.repo}. Read AGENTS.md and use the installed ctx skill at .agents/skills/ctx/SKILL.md. Start from existing .ctx/local/CONTINUE.md and .ctx/INDEX.md, load relevant context, and check current source for material claims. This is read-only: do not modify files, stage, commit, install, publish, delegate, or inspect parents or other repositories. You have read, grep, find, and ls tools only. Answer the question with source paths, evidence limits, and a concise list of documents used; do not create an audit file.\n\n${tasks[trial.task]}`;
 const args=['--mode','json','--print','--no-session','--offline','--no-extensions','--no-mcp','--no-skills','--no-context-files','--no-prompt-templates','--no-themes','--tools','read,grep,find,ls','--extension',scopeGuard,'--provider',trial.model.provider,'--model',trial.model.model];
 if(trial.model.thinking) args.push('--thinking',trial.model.thinking);
 args.push(prompt);
 const record={task:trial.task,layout:trial.layout,configuredModel:trial.model,input:trial.input,prompt,calls:[],messages:[],usage:[],providerModels:[],elapsedMs:null,runtime:null,before};
 const started=Date.now();let buffer='',stderr='',interrupted=null;
 console.log(JSON.stringify({stage:'starting',task:trial.task,model:trial.model.name,layout:trial.layout}));
 const child=spawn('pi',args,{cwd:trial.repo,env:{...process.env,CTX_RETRIEVAL_ROOT:trial.repo,PI_OFFLINE:'1',PI_TELEMETRY:'0'},stdio:['ignore','pipe','pipe']});
 const stop=reason=>{if(!interrupted){interrupted=reason;child.kill('SIGTERM');setTimeout(()=>child.kill('SIGKILL'),1500).unref();}};
 const timer=setTimeout(()=>stop('six-minute time limit'),360000);
 function event(e){
  if(e.type==='tool_execution_start'){
   record.calls.push({tool:e.toolName,arguments:e.args,call:e.toolCallId});
   if(record.calls.length>24) stop('twenty-four tool-call limit');
  }
  if(e.type==='tool_execution_end') {
   const call=record.calls.find(call=>call.call===e.toolCallId);
   if(call){call.isError=!!e.isError;call.returnedBytes=(e.result?.content||[]).filter(c=>c.type==='text').reduce((n,c)=>n+Buffer.byteLength(c.text),0);}
  }
  if(e.type==='message_end' && e.message?.role==='assistant'){
   const m=e.message;
   const text=(m.content||[]).filter(c=>c.type==='text').map(c=>c.text).join('\n');
   const hasTools=(m.content||[]).some(c=>c.type==='toolCall');
   if(text) record.messages.push({text,hasTools,stopReason:m.stopReason});
   if(m.usage) record.usage.push(m.usage);
   if(m.provider||m.model) record.providerModels.push({provider:m.provider,model:m.model});
   if(m.errorMessage) record.error=/connection error/i.test(String(m.errorMessage)) ? 'Connection error.' : 'Provider/model error (details omitted from saved public record)';
  }
 }
 child.stdout.on('data',chunk=>{buffer+=chunk;let line;while((line=buffer.indexOf('\n'))>=0){const raw=buffer.slice(0,line);buffer=buffer.slice(line+1);try{event(JSON.parse(raw));}catch{}}});
 child.stderr.on('data',chunk=>{stderr+=chunk; if(stderr.length>4000)stderr=stderr.slice(-4000);});
 const completion=await new Promise(resolve=>{child.on('error',error=>resolve({spawnError:error.message}));child.on('close',(code,signal)=>resolve({code,signal}));});
 clearTimeout(timer);
 record.elapsedMs=Date.now()-started;
 record.runtime={...completion,interrupted};
 record.final=record.messages.filter(m=>!m.hasTools && m.stopReason==='stop').at(-1)?.text || '';
 record.stderrBytes=Buffer.byteLength(stderr);
 record.stderrOmitted=true;
 record.after=await manifest(trial.repo);
 record.workspaceUnchanged=JSON.stringify(record.before)===JSON.stringify(record.after);
 record.toolOutputBytes=record.calls.reduce((n,c)=>n+(c.returnedBytes||0),0);
 record.contextOutputBytes=record.calls.filter(c=>(c.arguments?.path||'').includes('.ctx')).reduce((n,c)=>n+(c.returnedBytes||0),0);
 records.push(record);
 await writeFile(join(root,'public-results.json'),JSON.stringify({piVersion:'1.1.0',records},null,2));
 console.log(JSON.stringify({stage:'finished',task:trial.task,model:trial.model.name,layout:trial.layout,calls:record.calls.length,bytes:record.toolOutputBytes,elapsedMs:record.elapsedMs,final:!!record.final,runtime:record.runtime,unchanged:record.workspaceUnchanged,error:record.error}));
}
console.log(JSON.stringify({stage:'complete',trials:records.length}));
