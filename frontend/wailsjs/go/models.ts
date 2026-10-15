export namespace main {
	
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
	    locations: string[];
	    protocolPorts: Record<string, Array<number>>;
	    circumventCensorship: boolean;
	    largeTls: boolean;
	    internalDns: string;
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
	        this.locations = source["locations"];
	        this.protocolPorts = source["protocolPorts"];
	        this.circumventCensorship = source["circumventCensorship"];
	        this.largeTls = source["largeTls"];
	        this.internalDns = source["internalDns"];
	        this.selectionMode = source["selectionMode"];
	        this.testPing = source["testPing"];
	        this.testDownload = source["testDownload"];
	        this.testUpload = source["testUpload"];
	        this.speedWait = source["speedWait"];
	        this.timeout = source["timeout"];
	    }
	}
	export class TestResult {
	    ID: number;
	    Location: string;
	    City: string;
	    Country: string;
	    Protocol: string;
	    Port: number;
	    Status: string;
	    Duration: number;
	    Ping: number;
	    Download: number;
	    Upload: number;
	    ProfileName: string;
	    AntiCensorshipMode: string;
	    LargeTLS: boolean;
	    InternalDNS: string;
	    Timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Location = source["Location"];
	        this.City = source["City"];
	        this.Country = source["Country"];
	        this.Protocol = source["Protocol"];
	        this.Port = source["Port"];
	        this.Status = source["Status"];
	        this.Duration = source["Duration"];
	        this.Ping = source["Ping"];
	        this.Download = source["Download"];
	        this.Upload = source["Upload"];
	        this.ProfileName = source["ProfileName"];
	        this.AntiCensorshipMode = source["AntiCensorshipMode"];
	        this.LargeTLS = source["LargeTLS"];
	        this.InternalDNS = source["InternalDNS"];
	        this.Timestamp = source["Timestamp"];
	    }
	}

}

