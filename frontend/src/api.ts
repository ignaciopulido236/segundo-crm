export const fields = ['nombre','apellido','email','telefono','empresa','cargo','estado','origen','responsable','direccion','ciudad','pais','notas'] as const;
export type Field = typeof fields[number];
export type Draft = Record<Field,string>;
export type Client = Draft & {id:string;creado_en:string;actualizado_en:string};
export type Page = {data:Client[];meta:{total:number;limit:number;offset:number}};
export type Config = {url:string;configured:boolean;problem?:string};
export interface Bridge {
 SetSessionState?(dirty:boolean,pending:boolean):Promise<void>; Configuration():Promise<Config>; Configure(url:string,key:string):Promise<void>;
 List(q:string,estado:string,offset:number):Promise<Page>;Get(id:string):Promise<Client>;
 Create(data:Partial<Draft>):Promise<Client>;Update(id:string,data:Partial<Draft>):Promise<Client>;Delete(id:string):Promise<void>;
}
declare global {interface Window {go:{main:{App:Bridge}}}}
export const bridge:Bridge = new Proxy({} as Bridge,{get:(_,method:keyof Bridge)=>(...args:unknown[])=>{
 const app=window.go?.main?.App;
 if(!app)return Promise.reject(new Error('Abre PrimerCRM en Windows con Wails para conectar con la API.'));
 return (app[method] as (...args:unknown[])=>unknown)(...args);
}});
export const empty = ():Draft=>Object.fromEntries(fields.map(f=>[f,f==='estado'?'nuevo':''])) as Draft;
export const toDraft = (c:Client):Draft=>Object.fromEntries(fields.map(f=>[f,c[f]??''])) as Draft;
export const states=['nuevo','contactado','calificado','cliente','inactivo'];
