export namespace discovery {
	
	export class PortInfo {
	    port: number;
	    protocol: string;
	    process?: string;
	    pid?: number;
	
	    static createFrom(source: any = {}) {
	        return new PortInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.protocol = source["protocol"];
	        this.process = source["process"];
	        this.pid = source["pid"];
	    }
	}

}

