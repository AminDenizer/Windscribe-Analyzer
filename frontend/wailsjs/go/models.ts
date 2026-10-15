export namespace main {
	
	export class AmneziaConfigItem {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AmneziaConfigItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class LocationData {
	    location: string;
	    city: string;
	    region: string;
	    continent: string;
	    display: string;
	
	    static createFrom(source: any = {}) {
	        return new LocationData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.location = source["location"];
	        this.city = source["city"];
	        this.region = source["region"];
	        this.continent = source["continent"];
	        this.display = source["display"];
	    }
	}
	export class TestConfig {
	    selectedLocations: LocationData[];
	    protocolPorts: Record<string, Array<number>>;
	    protocolTweaks: boolean;
	    amneziaConfig: string;
	    serverRouting: string;
	    customSni: string;
	    largeTls: boolean;
	    internalDns: string;
	    isp: string;
	    isResume: boolean;
	    selectionMode: string;
	    testPing: boolean;
	    testDownload: boolean;
	    testUpload: boolean;
	    speedWait: number;
	    timeout: number;
	
	    static createFrom(source: any = {}) {
	        return new TestConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selectedLocations = this.convertValues(source["selectedLocations"], LocationData);
	        this.protocolPorts = source["protocolPorts"];
	        this.protocolTweaks = source["protocolTweaks"];
	        this.amneziaConfig = source["amneziaConfig"];
	        this.serverRouting = source["serverRouting"];
	        this.customSni = source["customSni"];
	        this.largeTls = source["largeTls"];
	        this.internalDns = source["internalDns"];
	        this.isp = source["isp"];
	        this.isResume = source["isResume"];
	        this.selectionMode = source["selectionMode"];
	        this.testPing = source["testPing"];
	        this.testDownload = source["testDownload"];
	        this.testUpload = source["testUpload"];
	        this.speedWait = source["speedWait"];
	        this.timeout = source["timeout"];
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
	export class TestResult {
	    id: number;
	    location: string;
	    city: string;
	    country: string;
	    continent: string;
	    protocol: string;
	    port: number;
	    status: string;
	    duration: number;
	    ping: number;
	    jitter: number;
	    download: number;
	    upload: number;
	    profileName: string;
	    protocolTweaks: boolean;
	    antiCensorshipMode: string;
	    serverRouting: string;
	    customSni: string;
	    largeTls: boolean;
	    internalDns: string;
	    isp: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.location = source["location"];
	        this.city = source["city"];
	        this.country = source["country"];
	        this.continent = source["continent"];
	        this.protocol = source["protocol"];
	        this.port = source["port"];
	        this.status = source["status"];
	        this.duration = source["duration"];
	        this.ping = source["ping"];
	        this.jitter = source["jitter"];
	        this.download = source["download"];
	        this.upload = source["upload"];
	        this.profileName = source["profileName"];
	        this.protocolTweaks = source["protocolTweaks"];
	        this.antiCensorshipMode = source["antiCensorshipMode"];
	        this.serverRouting = source["serverRouting"];
	        this.customSni = source["customSni"];
	        this.largeTls = source["largeTls"];
	        this.internalDns = source["internalDns"];
	        this.isp = source["isp"];
	        this.timestamp = source["timestamp"];
	    }
	}

}

