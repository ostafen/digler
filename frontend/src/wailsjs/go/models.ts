export namespace api {
	
	export class DeviceInfo {
	    name: string;
	    path: string;
	    model: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new DeviceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.model = source["model"];
	        this.size = source["size"];
	    }
	}
	export class FileInfo {
	    name: string;
	    ext: string;
	    offset: number;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ext = source["ext"];
	        this.offset = source["offset"];
	        this.size = source["size"];
	    }
	}
	export class RecoveryStatus {
	    progress: number;
	    recovered: number;
	    errors: number;
	
	    static createFrom(source: any = {}) {
	        return new RecoveryStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.progress = source["progress"];
	        this.recovered = source["recovered"];
	        this.errors = source["errors"];
	    }
	}
	export class ScanHistoryRecord {
	    id: string;
	    scanStartedAt: number;
	    sourcePath: string;
	    sourceType: string;
	    filesFound: number;
	    signaturesFound: number;
	    isMissing: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScanHistoryRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.scanStartedAt = source["scanStartedAt"];
	        this.sourcePath = source["sourcePath"];
	        this.sourceType = source["sourceType"];
	        this.filesFound = source["filesFound"];
	        this.signaturesFound = source["signaturesFound"];
	        this.isMissing = source["isMissing"];
	    }
	}
	export class ScanResultResponse {
	    files: FileInfo[];
	
	    static createFrom(source: any = {}) {
	        return new ScanResultResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = this.convertValues(source["files"], FileInfo);
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
	export class ScanStatusResponse {
	    status: string;
	    logs: string[];
	    progress: number;
	    signatures: number;
	    files: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.logs = source["logs"];
	        this.progress = source["progress"];
	        this.signatures = source["signatures"];
	        this.files = source["files"];
	    }
	}

}

export namespace app {
	
	export class FileDialogFilter {
	    name: string;
	    pattern: string;
	
	    static createFrom(source: any = {}) {
	        return new FileDialogFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pattern = source["pattern"];
	    }
	}

}

