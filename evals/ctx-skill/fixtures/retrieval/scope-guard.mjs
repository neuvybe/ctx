import { realpathSync } from 'node:fs';
import { isAbsolute, relative, resolve, sep } from 'node:path';
export default function(pi) {
  const root = realpathSync(process.env.CTX_RETRIEVAL_ROOT);
  pi.on('tool_call', event => {
    if (!['read','grep','find','ls'].includes(event.toolName)) return {block:true,reason:'Only read-only retrieval tools are authorized'};
    const path = resolve(root, event.input.path || '.');
    const inside = value => { const rel=relative(root,value); return !isAbsolute(rel) && rel !== '..' && !rel.startsWith('..'+sep) && !rel.split(sep).includes('.git'); };
    if (!inside(path)) return {block:true,reason:'Path is outside the authorized repository or inside Git internals'};
    try { if(!inside(realpathSync(path))) return {block:true,reason:'Resolved path leaves the authorized repository'}; }
    catch(error) { return {block:true,reason:'The requested safe path does not exist: '+error.code}; }
  });
}
