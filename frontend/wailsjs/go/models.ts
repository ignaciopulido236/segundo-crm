export namespace crm {
	
	export class Client {
	    id: string;
	    nombre: string;
	    apellido: string;
	    email: string;
	    telefono: string;
	    empresa: string;
	    cargo: string;
	    estado: string;
	    origen: string;
	    responsable: string;
	    direccion: string;
	    ciudad: string;
	    pais: string;
	    notas: string;
	    creado_en: string;
	    actualizado_en: string;
	
	    static createFrom(source: any = {}) {
	        return new Client(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nombre = source["nombre"];
	        this.apellido = source["apellido"];
	        this.email = source["email"];
	        this.telefono = source["telefono"];
	        this.empresa = source["empresa"];
	        this.cargo = source["cargo"];
	        this.estado = source["estado"];
	        this.origen = source["origen"];
	        this.responsable = source["responsable"];
	        this.direccion = source["direccion"];
	        this.ciudad = source["ciudad"];
	        this.pais = source["pais"];
	        this.notas = source["notas"];
	        this.creado_en = source["creado_en"];
	        this.actualizado_en = source["actualizado_en"];
	    }
	}
	export class Meta {
	    total: number;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new Meta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class Page {
	    data: Client[];
	    meta: Meta;
	
	    static createFrom(source: any = {}) {
	        return new Page(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], Client);
	        this.meta = this.convertValues(source["meta"], Meta);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PublicConfig {
	    url: string;
	    configured: boolean;
	    problem?: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.configured = source["configured"];
	        this.problem = source["problem"];
	    }
	}

}

