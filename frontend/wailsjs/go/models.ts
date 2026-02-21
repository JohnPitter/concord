export namespace auth {
	
	export class User {
	    id: string;
	    github_id: number;
	    username: string;
	    display_name: string;
	    avatar_url: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.github_id = source["github_id"];
	        this.username = source["username"];
	        this.display_name = source["display_name"];
	        this.avatar_url = source["avatar_url"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class AuthState {
	    authenticated: boolean;
	    user?: User;
	    access_token?: string;
	    expires_at?: number;
	
	    static createFrom(source: any = {}) {
	        return new AuthState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.user = this.convertValues(source["user"], User);
	        this.access_token = source["access_token"];
	        this.expires_at = source["expires_at"];
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
	export class DeviceCodeResponse {
	    device_code: string;
	    user_code: string;
	    verification_uri: string;
	    expires_in: number;
	    interval: number;
	
	    static createFrom(source: any = {}) {
	        return new DeviceCodeResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device_code = source["device_code"];
	        this.user_code = source["user_code"];
	        this.verification_uri = source["verification_uri"];
	        this.expires_in = source["expires_in"];
	        this.interval = source["interval"];
	    }
	}

}

export namespace chat {
	
	export class Message {
	    id: string;
	    channel_id: string;
	    author_id: string;
	    content: string;
	    type: string;
	    edited_at?: string;
	    created_at: string;
	    author_name?: string;
	    author_avatar?: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.channel_id = source["channel_id"];
	        this.author_id = source["author_id"];
	        this.content = source["content"];
	        this.type = source["type"];
	        this.edited_at = source["edited_at"];
	        this.created_at = source["created_at"];
	        this.author_name = source["author_name"];
	        this.author_avatar = source["author_avatar"];
	    }
	}
	export class SearchResult {
	    id: string;
	    channel_id: string;
	    author_id: string;
	    content: string;
	    type: string;
	    edited_at?: string;
	    created_at: string;
	    author_name?: string;
	    author_avatar?: string;
	    snippet: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.channel_id = source["channel_id"];
	        this.author_id = source["author_id"];
	        this.content = source["content"];
	        this.type = source["type"];
	        this.edited_at = source["edited_at"];
	        this.created_at = source["created_at"];
	        this.author_name = source["author_name"];
	        this.author_avatar = source["author_avatar"];
	        this.snippet = source["snippet"];
	    }
	}

}

export namespace files {
	
	export class Attachment {
	    id: string;
	    message_id: string;
	    filename: string;
	    size_bytes: number;
	    mime_type: string;
	    hash: string;
	    local_path?: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Attachment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.message_id = source["message_id"];
	        this.filename = source["filename"];
	        this.size_bytes = source["size_bytes"];
	        this.mime_type = source["mime_type"];
	        this.hash = source["hash"];
	        this.local_path = source["local_path"];
	        this.created_at = source["created_at"];
	    }
	}

}

export namespace observability {
	
	export class ComponentHealth {
	    name: string;
	    status: string;
	    message?: string;
	    error?: string;
	    timestamp: string;
	    duration_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new ComponentHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.error = source["error"];
	        this.timestamp = source["timestamp"];
	        this.duration_ms = source["duration_ms"];
	    }
	}
	export class Health {
	    status: string;
	    timestamp: string;
	    components: Record<string, ComponentHealth>;
	    version: string;
	    uptime_seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new Health(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.timestamp = source["timestamp"];
	        this.components = this.convertValues(source["components"], ComponentHealth, true);
	        this.version = source["version"];
	        this.uptime_seconds = source["uptime_seconds"];
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

}

export namespace server {
	
	export class Channel {
	    id: string;
	    server_id: string;
	    name: string;
	    type: string;
	    position: number;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Channel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.server_id = source["server_id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.position = source["position"];
	        this.created_at = source["created_at"];
	    }
	}
	export class InviteInfo {
	    server_id: string;
	    server_name: string;
	    invite_code: string;
	    member_count: number;
	
	    static createFrom(source: any = {}) {
	        return new InviteInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server_id = source["server_id"];
	        this.server_name = source["server_name"];
	        this.invite_code = source["invite_code"];
	        this.member_count = source["member_count"];
	    }
	}
	export class Member {
	    server_id: string;
	    user_id: string;
	    username: string;
	    avatar_url: string;
	    role: string;
	    joined_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Member(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server_id = source["server_id"];
	        this.user_id = source["user_id"];
	        this.username = source["username"];
	        this.avatar_url = source["avatar_url"];
	        this.role = source["role"];
	        this.joined_at = source["joined_at"];
	    }
	}
	export class Server {
	    id: string;
	    name: string;
	    icon_url: string;
	    owner_id: string;
	    invite_code: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon_url = source["icon_url"];
	        this.owner_id = source["owner_id"];
	        this.invite_code = source["invite_code"];
	        this.created_at = source["created_at"];
	    }
	}

}

export namespace translation {
	
	export class Status {
	    enabled: boolean;
	    source_lang: string;
	    target_lang: string;
	    circuit_breaker_open: boolean;
	    cache_entries: number;
	    pipeline_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.source_lang = source["source_lang"];
	        this.target_lang = source["target_lang"];
	        this.circuit_breaker_open = source["circuit_breaker_open"];
	        this.cache_entries = source["cache_entries"];
	        this.pipeline_active = source["pipeline_active"];
	    }
	}

}

export namespace version {
	
	export class Info {
	    version: string;
	    git_commit: string;
	    build_date: string;
	    go_version: string;
	    platform: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.git_commit = source["git_commit"];
	        this.build_date = source["build_date"];
	        this.go_version = source["go_version"];
	        this.platform = source["platform"];
	    }
	}

}

export namespace voice {
	
	export class SpeakerInfo {
	    peer_id: string;
	    user_id: string;
	    username: string;
	    volume: number;
	    speaking: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SpeakerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.peer_id = source["peer_id"];
	        this.user_id = source["user_id"];
	        this.username = source["username"];
	        this.volume = source["volume"];
	        this.speaking = source["speaking"];
	    }
	}
	export class VoiceStatus {
	    state: string;
	    channel_id: string;
	    muted: boolean;
	    deafened: boolean;
	    peer_count: number;
	    speakers: SpeakerInfo[];
	
	    static createFrom(source: any = {}) {
	        return new VoiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.channel_id = source["channel_id"];
	        this.muted = source["muted"];
	        this.deafened = source["deafened"];
	        this.peer_count = source["peer_count"];
	        this.speakers = this.convertValues(source["speakers"], SpeakerInfo);
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

}

