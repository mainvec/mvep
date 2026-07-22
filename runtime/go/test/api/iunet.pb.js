/*eslint-disable block-scoped-var, id-length, no-control-regex, no-magic-numbers, no-prototype-builtins, no-redeclare, no-shadow, no-var, sort-vars*/
import * as $protobuf from "protobufjs/minimal";

// Common aliases
const $Reader = $protobuf.Reader, $Writer = $protobuf.Writer, $util = $protobuf.util;

// Exported root namespace
const $root = $protobuf.roots["default"] || ($protobuf.roots["default"] = {});

export const mainvec = $root.mainvec = (() => {

    /**
     * Namespace mainvec.
     * @exports mainvec
     * @namespace
     */
    const mainvec = {};

    mainvec.iunet = (function() {

        /**
         * Namespace iunet.
         * @memberof mainvec
         * @namespace
         */
        const iunet = {};

        iunet.Serverlet = (function() {

            /**
             * Properties of a Serverlet.
             * @memberof mainvec.iunet
             * @interface IServerlet
             * @property {string|null} [name] Serverlet name
             * @property {string|null} [uuid] Serverlet uuid
             * @property {string|null} [hubUUID] Serverlet hubUUID
             * @property {string|null} [desc] Serverlet desc
             * @property {number|null} [port] Serverlet port
             * @property {string|null} [serverletAddress] Serverlet serverletAddress
             * @property {boolean|null} [disabled] Serverlet disabled
             * @property {boolean|null} [manualStartup] Serverlet manualStartup
             * @property {number|null} [connState] Serverlet connState
             */

            /**
             * Constructs a new Serverlet.
             * @memberof mainvec.iunet
             * @classdesc Represents a Serverlet.
             * @implements IServerlet
             * @constructor
             * @param {mainvec.iunet.IServerlet=} [properties] Properties to set
             */
            function Serverlet(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Serverlet name.
             * @member {string} name
             * @memberof mainvec.iunet.Serverlet
             * @instance
             */
            Serverlet.prototype.name = "";

            /**
             * Serverlet uuid.
             * @member {string} uuid
             * @memberof mainvec.iunet.Serverlet
             * @instance
             */
            Serverlet.prototype.uuid = "";

            /**
             * Serverlet hubUUID.
             * @member {string} hubUUID
             * @memberof mainvec.iunet.Serverlet
             * @instance
             */
            Serverlet.prototype.hubUUID = "";

            /**
             * Serverlet desc.
             * @member {string} desc
             * @memberof mainvec.iunet.Serverlet
             * @instance
             */
            Serverlet.prototype.desc = "";

            /**
             * Serverlet port.
             * @member {number} port
             * @memberof mainvec.iunet.Serverlet
             * @instance
             */
            Serverlet.prototype.port = 0;

            /**
             * Serverlet serverletAddress.
             * @member {string} serverletAddress
             * @memberof mainvec.iunet.Serverlet
             * @instance
             */
            Serverlet.prototype.serverletAddress = "";

            /**
             * Serverlet disabled.
             * @member {boolean} disabled
             * @memberof mainvec.iunet.Serverlet
             * @instance
             */
            Serverlet.prototype.disabled = false;

            /**
             * Serverlet manualStartup.
             * @member {boolean} manualStartup
             * @memberof mainvec.iunet.Serverlet
             * @instance
             */
            Serverlet.prototype.manualStartup = false;

            /**
             * Serverlet connState.
             * @member {number} connState
             * @memberof mainvec.iunet.Serverlet
             * @instance
             */
            Serverlet.prototype.connState = 0;

            /**
             * Creates a new Serverlet instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.Serverlet
             * @static
             * @param {mainvec.iunet.IServerlet=} [properties] Properties to set
             * @returns {mainvec.iunet.Serverlet} Serverlet instance
             */
            Serverlet.create = function create(properties) {
                return new Serverlet(properties);
            };

            /**
             * Encodes the specified Serverlet message. Does not implicitly {@link mainvec.iunet.Serverlet.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.Serverlet
             * @static
             * @param {mainvec.iunet.IServerlet} message Serverlet message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Serverlet.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.uuid != null && Object.hasOwnProperty.call(message, "uuid"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.uuid);
                if (message.hubUUID != null && Object.hasOwnProperty.call(message, "hubUUID"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.hubUUID);
                if (message.desc != null && Object.hasOwnProperty.call(message, "desc"))
                    writer.uint32(/* id 5, wireType 2 =*/42).string(message.desc);
                if (message.port != null && Object.hasOwnProperty.call(message, "port"))
                    writer.uint32(/* id 10, wireType 0 =*/80).int32(message.port);
                if (message.serverletAddress != null && Object.hasOwnProperty.call(message, "serverletAddress"))
                    writer.uint32(/* id 12, wireType 2 =*/98).string(message.serverletAddress);
                if (message.disabled != null && Object.hasOwnProperty.call(message, "disabled"))
                    writer.uint32(/* id 14, wireType 0 =*/112).bool(message.disabled);
                if (message.manualStartup != null && Object.hasOwnProperty.call(message, "manualStartup"))
                    writer.uint32(/* id 15, wireType 0 =*/120).bool(message.manualStartup);
                if (message.connState != null && Object.hasOwnProperty.call(message, "connState"))
                    writer.uint32(/* id 16, wireType 0 =*/128).int32(message.connState);
                return writer;
            };

            /**
             * Encodes the specified Serverlet message, length delimited. Does not implicitly {@link mainvec.iunet.Serverlet.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.Serverlet
             * @static
             * @param {mainvec.iunet.IServerlet} message Serverlet message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Serverlet.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a Serverlet message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.Serverlet
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.Serverlet} Serverlet
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Serverlet.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.Serverlet();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.uuid = reader.string();
                            break;
                        }
                    case 3: {
                            message.hubUUID = reader.string();
                            break;
                        }
                    case 5: {
                            message.desc = reader.string();
                            break;
                        }
                    case 10: {
                            message.port = reader.int32();
                            break;
                        }
                    case 12: {
                            message.serverletAddress = reader.string();
                            break;
                        }
                    case 14: {
                            message.disabled = reader.bool();
                            break;
                        }
                    case 15: {
                            message.manualStartup = reader.bool();
                            break;
                        }
                    case 16: {
                            message.connState = reader.int32();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a Serverlet message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.Serverlet
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.Serverlet} Serverlet
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Serverlet.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a Serverlet message.
             * @function verify
             * @memberof mainvec.iunet.Serverlet
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Serverlet.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    if (!$util.isString(message.uuid))
                        return "uuid: string expected";
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    if (!$util.isString(message.hubUUID))
                        return "hubUUID: string expected";
                if (message.desc != null && message.hasOwnProperty("desc"))
                    if (!$util.isString(message.desc))
                        return "desc: string expected";
                if (message.port != null && message.hasOwnProperty("port"))
                    if (!$util.isInteger(message.port))
                        return "port: integer expected";
                if (message.serverletAddress != null && message.hasOwnProperty("serverletAddress"))
                    if (!$util.isString(message.serverletAddress))
                        return "serverletAddress: string expected";
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    if (typeof message.disabled !== "boolean")
                        return "disabled: boolean expected";
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    if (typeof message.manualStartup !== "boolean")
                        return "manualStartup: boolean expected";
                if (message.connState != null && message.hasOwnProperty("connState"))
                    if (!$util.isInteger(message.connState))
                        return "connState: integer expected";
                return null;
            };

            /**
             * Creates a Serverlet message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.Serverlet
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.Serverlet} Serverlet
             */
            Serverlet.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.Serverlet)
                    return object;
                let message = new $root.mainvec.iunet.Serverlet();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.uuid != null)
                    message.uuid = String(object.uuid);
                if (object.hubUUID != null)
                    message.hubUUID = String(object.hubUUID);
                if (object.desc != null)
                    message.desc = String(object.desc);
                if (object.port != null)
                    message.port = object.port | 0;
                if (object.serverletAddress != null)
                    message.serverletAddress = String(object.serverletAddress);
                if (object.disabled != null)
                    message.disabled = Boolean(object.disabled);
                if (object.manualStartup != null)
                    message.manualStartup = Boolean(object.manualStartup);
                if (object.connState != null)
                    message.connState = object.connState | 0;
                return message;
            };

            /**
             * Creates a plain object from a Serverlet message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.Serverlet
             * @static
             * @param {mainvec.iunet.Serverlet} message Serverlet
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Serverlet.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.uuid = "";
                    object.hubUUID = "";
                    object.desc = "";
                    object.port = 0;
                    object.serverletAddress = "";
                    object.disabled = false;
                    object.manualStartup = false;
                    object.connState = 0;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    object.uuid = message.uuid;
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    object.hubUUID = message.hubUUID;
                if (message.desc != null && message.hasOwnProperty("desc"))
                    object.desc = message.desc;
                if (message.port != null && message.hasOwnProperty("port"))
                    object.port = message.port;
                if (message.serverletAddress != null && message.hasOwnProperty("serverletAddress"))
                    object.serverletAddress = message.serverletAddress;
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    object.disabled = message.disabled;
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    object.manualStartup = message.manualStartup;
                if (message.connState != null && message.hasOwnProperty("connState"))
                    object.connState = message.connState;
                return object;
            };

            /**
             * Converts this Serverlet to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.Serverlet
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Serverlet.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for Serverlet
             * @function getTypeUrl
             * @memberof mainvec.iunet.Serverlet
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            Serverlet.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.Serverlet";
            };

            return Serverlet;
        })();

        iunet.Clientlet = (function() {

            /**
             * Properties of a Clientlet.
             * @memberof mainvec.iunet
             * @interface IClientlet
             * @property {string|null} [name] Clientlet name
             * @property {string|null} [uuid] Clientlet uuid
             * @property {string|null} [hubUUID] Clientlet hubUUID
             * @property {string|null} [listeningAddr] Clientlet listeningAddr
             * @property {string|null} [desc] Clientlet desc
             * @property {string|null} [serverletAddress] Clientlet serverletAddress
             * @property {number|null} [clientletPort] Clientlet clientletPort
             * @property {boolean|null} [disabled] Clientlet disabled
             * @property {boolean|null} [manualStartup] Clientlet manualStartup
             * @property {number|null} [connState] Clientlet connState
             */

            /**
             * Constructs a new Clientlet.
             * @memberof mainvec.iunet
             * @classdesc Represents a Clientlet.
             * @implements IClientlet
             * @constructor
             * @param {mainvec.iunet.IClientlet=} [properties] Properties to set
             */
            function Clientlet(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Clientlet name.
             * @member {string} name
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.name = "";

            /**
             * Clientlet uuid.
             * @member {string} uuid
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.uuid = "";

            /**
             * Clientlet hubUUID.
             * @member {string} hubUUID
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.hubUUID = "";

            /**
             * Clientlet listeningAddr.
             * @member {string} listeningAddr
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.listeningAddr = "";

            /**
             * Clientlet desc.
             * @member {string} desc
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.desc = "";

            /**
             * Clientlet serverletAddress.
             * @member {string} serverletAddress
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.serverletAddress = "";

            /**
             * Clientlet clientletPort.
             * @member {number} clientletPort
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.clientletPort = 0;

            /**
             * Clientlet disabled.
             * @member {boolean} disabled
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.disabled = false;

            /**
             * Clientlet manualStartup.
             * @member {boolean} manualStartup
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.manualStartup = false;

            /**
             * Clientlet connState.
             * @member {number} connState
             * @memberof mainvec.iunet.Clientlet
             * @instance
             */
            Clientlet.prototype.connState = 0;

            /**
             * Creates a new Clientlet instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.Clientlet
             * @static
             * @param {mainvec.iunet.IClientlet=} [properties] Properties to set
             * @returns {mainvec.iunet.Clientlet} Clientlet instance
             */
            Clientlet.create = function create(properties) {
                return new Clientlet(properties);
            };

            /**
             * Encodes the specified Clientlet message. Does not implicitly {@link mainvec.iunet.Clientlet.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.Clientlet
             * @static
             * @param {mainvec.iunet.IClientlet} message Clientlet message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Clientlet.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.uuid != null && Object.hasOwnProperty.call(message, "uuid"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.uuid);
                if (message.hubUUID != null && Object.hasOwnProperty.call(message, "hubUUID"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.hubUUID);
                if (message.listeningAddr != null && Object.hasOwnProperty.call(message, "listeningAddr"))
                    writer.uint32(/* id 5, wireType 2 =*/42).string(message.listeningAddr);
                if (message.desc != null && Object.hasOwnProperty.call(message, "desc"))
                    writer.uint32(/* id 7, wireType 2 =*/58).string(message.desc);
                if (message.serverletAddress != null && Object.hasOwnProperty.call(message, "serverletAddress"))
                    writer.uint32(/* id 10, wireType 2 =*/82).string(message.serverletAddress);
                if (message.clientletPort != null && Object.hasOwnProperty.call(message, "clientletPort"))
                    writer.uint32(/* id 12, wireType 0 =*/96).int32(message.clientletPort);
                if (message.disabled != null && Object.hasOwnProperty.call(message, "disabled"))
                    writer.uint32(/* id 14, wireType 0 =*/112).bool(message.disabled);
                if (message.manualStartup != null && Object.hasOwnProperty.call(message, "manualStartup"))
                    writer.uint32(/* id 15, wireType 0 =*/120).bool(message.manualStartup);
                if (message.connState != null && Object.hasOwnProperty.call(message, "connState"))
                    writer.uint32(/* id 16, wireType 0 =*/128).int32(message.connState);
                return writer;
            };

            /**
             * Encodes the specified Clientlet message, length delimited. Does not implicitly {@link mainvec.iunet.Clientlet.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.Clientlet
             * @static
             * @param {mainvec.iunet.IClientlet} message Clientlet message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Clientlet.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a Clientlet message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.Clientlet
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.Clientlet} Clientlet
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Clientlet.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.Clientlet();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.uuid = reader.string();
                            break;
                        }
                    case 3: {
                            message.hubUUID = reader.string();
                            break;
                        }
                    case 5: {
                            message.listeningAddr = reader.string();
                            break;
                        }
                    case 7: {
                            message.desc = reader.string();
                            break;
                        }
                    case 10: {
                            message.serverletAddress = reader.string();
                            break;
                        }
                    case 12: {
                            message.clientletPort = reader.int32();
                            break;
                        }
                    case 14: {
                            message.disabled = reader.bool();
                            break;
                        }
                    case 15: {
                            message.manualStartup = reader.bool();
                            break;
                        }
                    case 16: {
                            message.connState = reader.int32();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a Clientlet message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.Clientlet
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.Clientlet} Clientlet
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Clientlet.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a Clientlet message.
             * @function verify
             * @memberof mainvec.iunet.Clientlet
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Clientlet.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    if (!$util.isString(message.uuid))
                        return "uuid: string expected";
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    if (!$util.isString(message.hubUUID))
                        return "hubUUID: string expected";
                if (message.listeningAddr != null && message.hasOwnProperty("listeningAddr"))
                    if (!$util.isString(message.listeningAddr))
                        return "listeningAddr: string expected";
                if (message.desc != null && message.hasOwnProperty("desc"))
                    if (!$util.isString(message.desc))
                        return "desc: string expected";
                if (message.serverletAddress != null && message.hasOwnProperty("serverletAddress"))
                    if (!$util.isString(message.serverletAddress))
                        return "serverletAddress: string expected";
                if (message.clientletPort != null && message.hasOwnProperty("clientletPort"))
                    if (!$util.isInteger(message.clientletPort))
                        return "clientletPort: integer expected";
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    if (typeof message.disabled !== "boolean")
                        return "disabled: boolean expected";
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    if (typeof message.manualStartup !== "boolean")
                        return "manualStartup: boolean expected";
                if (message.connState != null && message.hasOwnProperty("connState"))
                    if (!$util.isInteger(message.connState))
                        return "connState: integer expected";
                return null;
            };

            /**
             * Creates a Clientlet message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.Clientlet
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.Clientlet} Clientlet
             */
            Clientlet.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.Clientlet)
                    return object;
                let message = new $root.mainvec.iunet.Clientlet();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.uuid != null)
                    message.uuid = String(object.uuid);
                if (object.hubUUID != null)
                    message.hubUUID = String(object.hubUUID);
                if (object.listeningAddr != null)
                    message.listeningAddr = String(object.listeningAddr);
                if (object.desc != null)
                    message.desc = String(object.desc);
                if (object.serverletAddress != null)
                    message.serverletAddress = String(object.serverletAddress);
                if (object.clientletPort != null)
                    message.clientletPort = object.clientletPort | 0;
                if (object.disabled != null)
                    message.disabled = Boolean(object.disabled);
                if (object.manualStartup != null)
                    message.manualStartup = Boolean(object.manualStartup);
                if (object.connState != null)
                    message.connState = object.connState | 0;
                return message;
            };

            /**
             * Creates a plain object from a Clientlet message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.Clientlet
             * @static
             * @param {mainvec.iunet.Clientlet} message Clientlet
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Clientlet.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.uuid = "";
                    object.hubUUID = "";
                    object.listeningAddr = "";
                    object.desc = "";
                    object.serverletAddress = "";
                    object.clientletPort = 0;
                    object.disabled = false;
                    object.manualStartup = false;
                    object.connState = 0;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    object.uuid = message.uuid;
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    object.hubUUID = message.hubUUID;
                if (message.listeningAddr != null && message.hasOwnProperty("listeningAddr"))
                    object.listeningAddr = message.listeningAddr;
                if (message.desc != null && message.hasOwnProperty("desc"))
                    object.desc = message.desc;
                if (message.serverletAddress != null && message.hasOwnProperty("serverletAddress"))
                    object.serverletAddress = message.serverletAddress;
                if (message.clientletPort != null && message.hasOwnProperty("clientletPort"))
                    object.clientletPort = message.clientletPort;
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    object.disabled = message.disabled;
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    object.manualStartup = message.manualStartup;
                if (message.connState != null && message.hasOwnProperty("connState"))
                    object.connState = message.connState;
                return object;
            };

            /**
             * Converts this Clientlet to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.Clientlet
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Clientlet.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for Clientlet
             * @function getTypeUrl
             * @memberof mainvec.iunet.Clientlet
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            Clientlet.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.Clientlet";
            };

            return Clientlet;
        })();

        iunet.Hub = (function() {

            /**
             * Properties of a Hub.
             * @memberof mainvec.iunet
             * @interface IHub
             * @property {string|null} [name] Hub name
             * @property {string|null} [uuid] Hub uuid
             * @property {string|null} [desc] Hub desc
             * @property {Array.<mainvec.iunet.IServerlet>|null} [serverlets] Hub serverlets
             * @property {Array.<mainvec.iunet.IClientlet>|null} [clientlets] Hub clientlets
             * @property {boolean|null} [disabled] Hub disabled
             * @property {boolean|null} [manualStartup] Hub manualStartup
             */

            /**
             * Constructs a new Hub.
             * @memberof mainvec.iunet
             * @classdesc Represents a Hub.
             * @implements IHub
             * @constructor
             * @param {mainvec.iunet.IHub=} [properties] Properties to set
             */
            function Hub(properties) {
                this.serverlets = [];
                this.clientlets = [];
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Hub name.
             * @member {string} name
             * @memberof mainvec.iunet.Hub
             * @instance
             */
            Hub.prototype.name = "";

            /**
             * Hub uuid.
             * @member {string} uuid
             * @memberof mainvec.iunet.Hub
             * @instance
             */
            Hub.prototype.uuid = "";

            /**
             * Hub desc.
             * @member {string} desc
             * @memberof mainvec.iunet.Hub
             * @instance
             */
            Hub.prototype.desc = "";

            /**
             * Hub serverlets.
             * @member {Array.<mainvec.iunet.IServerlet>} serverlets
             * @memberof mainvec.iunet.Hub
             * @instance
             */
            Hub.prototype.serverlets = $util.emptyArray;

            /**
             * Hub clientlets.
             * @member {Array.<mainvec.iunet.IClientlet>} clientlets
             * @memberof mainvec.iunet.Hub
             * @instance
             */
            Hub.prototype.clientlets = $util.emptyArray;

            /**
             * Hub disabled.
             * @member {boolean} disabled
             * @memberof mainvec.iunet.Hub
             * @instance
             */
            Hub.prototype.disabled = false;

            /**
             * Hub manualStartup.
             * @member {boolean} manualStartup
             * @memberof mainvec.iunet.Hub
             * @instance
             */
            Hub.prototype.manualStartup = false;

            /**
             * Creates a new Hub instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.Hub
             * @static
             * @param {mainvec.iunet.IHub=} [properties] Properties to set
             * @returns {mainvec.iunet.Hub} Hub instance
             */
            Hub.create = function create(properties) {
                return new Hub(properties);
            };

            /**
             * Encodes the specified Hub message. Does not implicitly {@link mainvec.iunet.Hub.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.Hub
             * @static
             * @param {mainvec.iunet.IHub} message Hub message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Hub.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.uuid != null && Object.hasOwnProperty.call(message, "uuid"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.uuid);
                if (message.desc != null && Object.hasOwnProperty.call(message, "desc"))
                    writer.uint32(/* id 4, wireType 2 =*/34).string(message.desc);
                if (message.serverlets != null && message.serverlets.length)
                    for (let i = 0; i < message.serverlets.length; ++i)
                        $root.mainvec.iunet.Serverlet.encode(message.serverlets[i], writer.uint32(/* id 10, wireType 2 =*/82).fork()).ldelim();
                if (message.clientlets != null && message.clientlets.length)
                    for (let i = 0; i < message.clientlets.length; ++i)
                        $root.mainvec.iunet.Clientlet.encode(message.clientlets[i], writer.uint32(/* id 11, wireType 2 =*/90).fork()).ldelim();
                if (message.disabled != null && Object.hasOwnProperty.call(message, "disabled"))
                    writer.uint32(/* id 14, wireType 0 =*/112).bool(message.disabled);
                if (message.manualStartup != null && Object.hasOwnProperty.call(message, "manualStartup"))
                    writer.uint32(/* id 15, wireType 0 =*/120).bool(message.manualStartup);
                return writer;
            };

            /**
             * Encodes the specified Hub message, length delimited. Does not implicitly {@link mainvec.iunet.Hub.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.Hub
             * @static
             * @param {mainvec.iunet.IHub} message Hub message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Hub.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a Hub message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.Hub
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.Hub} Hub
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Hub.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.Hub();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.uuid = reader.string();
                            break;
                        }
                    case 4: {
                            message.desc = reader.string();
                            break;
                        }
                    case 10: {
                            if (!(message.serverlets && message.serverlets.length))
                                message.serverlets = [];
                            message.serverlets.push($root.mainvec.iunet.Serverlet.decode(reader, reader.uint32()));
                            break;
                        }
                    case 11: {
                            if (!(message.clientlets && message.clientlets.length))
                                message.clientlets = [];
                            message.clientlets.push($root.mainvec.iunet.Clientlet.decode(reader, reader.uint32()));
                            break;
                        }
                    case 14: {
                            message.disabled = reader.bool();
                            break;
                        }
                    case 15: {
                            message.manualStartup = reader.bool();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a Hub message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.Hub
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.Hub} Hub
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Hub.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a Hub message.
             * @function verify
             * @memberof mainvec.iunet.Hub
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Hub.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    if (!$util.isString(message.uuid))
                        return "uuid: string expected";
                if (message.desc != null && message.hasOwnProperty("desc"))
                    if (!$util.isString(message.desc))
                        return "desc: string expected";
                if (message.serverlets != null && message.hasOwnProperty("serverlets")) {
                    if (!Array.isArray(message.serverlets))
                        return "serverlets: array expected";
                    for (let i = 0; i < message.serverlets.length; ++i) {
                        let error = $root.mainvec.iunet.Serverlet.verify(message.serverlets[i]);
                        if (error)
                            return "serverlets." + error;
                    }
                }
                if (message.clientlets != null && message.hasOwnProperty("clientlets")) {
                    if (!Array.isArray(message.clientlets))
                        return "clientlets: array expected";
                    for (let i = 0; i < message.clientlets.length; ++i) {
                        let error = $root.mainvec.iunet.Clientlet.verify(message.clientlets[i]);
                        if (error)
                            return "clientlets." + error;
                    }
                }
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    if (typeof message.disabled !== "boolean")
                        return "disabled: boolean expected";
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    if (typeof message.manualStartup !== "boolean")
                        return "manualStartup: boolean expected";
                return null;
            };

            /**
             * Creates a Hub message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.Hub
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.Hub} Hub
             */
            Hub.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.Hub)
                    return object;
                let message = new $root.mainvec.iunet.Hub();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.uuid != null)
                    message.uuid = String(object.uuid);
                if (object.desc != null)
                    message.desc = String(object.desc);
                if (object.serverlets) {
                    if (!Array.isArray(object.serverlets))
                        throw TypeError(".mainvec.iunet.Hub.serverlets: array expected");
                    message.serverlets = [];
                    for (let i = 0; i < object.serverlets.length; ++i) {
                        if (typeof object.serverlets[i] !== "object")
                            throw TypeError(".mainvec.iunet.Hub.serverlets: object expected");
                        message.serverlets[i] = $root.mainvec.iunet.Serverlet.fromObject(object.serverlets[i]);
                    }
                }
                if (object.clientlets) {
                    if (!Array.isArray(object.clientlets))
                        throw TypeError(".mainvec.iunet.Hub.clientlets: array expected");
                    message.clientlets = [];
                    for (let i = 0; i < object.clientlets.length; ++i) {
                        if (typeof object.clientlets[i] !== "object")
                            throw TypeError(".mainvec.iunet.Hub.clientlets: object expected");
                        message.clientlets[i] = $root.mainvec.iunet.Clientlet.fromObject(object.clientlets[i]);
                    }
                }
                if (object.disabled != null)
                    message.disabled = Boolean(object.disabled);
                if (object.manualStartup != null)
                    message.manualStartup = Boolean(object.manualStartup);
                return message;
            };

            /**
             * Creates a plain object from a Hub message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.Hub
             * @static
             * @param {mainvec.iunet.Hub} message Hub
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Hub.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.arrays || options.defaults) {
                    object.serverlets = [];
                    object.clientlets = [];
                }
                if (options.defaults) {
                    object.name = "";
                    object.uuid = "";
                    object.desc = "";
                    object.disabled = false;
                    object.manualStartup = false;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    object.uuid = message.uuid;
                if (message.desc != null && message.hasOwnProperty("desc"))
                    object.desc = message.desc;
                if (message.serverlets && message.serverlets.length) {
                    object.serverlets = [];
                    for (let j = 0; j < message.serverlets.length; ++j)
                        object.serverlets[j] = $root.mainvec.iunet.Serverlet.toObject(message.serverlets[j], options);
                }
                if (message.clientlets && message.clientlets.length) {
                    object.clientlets = [];
                    for (let j = 0; j < message.clientlets.length; ++j)
                        object.clientlets[j] = $root.mainvec.iunet.Clientlet.toObject(message.clientlets[j], options);
                }
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    object.disabled = message.disabled;
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    object.manualStartup = message.manualStartup;
                return object;
            };

            /**
             * Converts this Hub to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.Hub
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Hub.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for Hub
             * @function getTypeUrl
             * @memberof mainvec.iunet.Hub
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            Hub.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.Hub";
            };

            return Hub;
        })();

        iunet.Broker = (function() {

            /**
             * Properties of a Broker.
             * @memberof mainvec.iunet
             * @interface IBroker
             * @property {string|null} [name] Broker name
             * @property {string|null} [uuid] Broker uuid
             * @property {string|null} [desc] Broker desc
             * @property {string|null} [type] Broker type
             * @property {string|null} [connParams] Broker connParams
             * @property {string|null} [authParams] Broker authParams
             * @property {boolean|null} [disabled] Broker disabled
             */

            /**
             * Constructs a new Broker.
             * @memberof mainvec.iunet
             * @classdesc Represents a Broker.
             * @implements IBroker
             * @constructor
             * @param {mainvec.iunet.IBroker=} [properties] Properties to set
             */
            function Broker(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Broker name.
             * @member {string} name
             * @memberof mainvec.iunet.Broker
             * @instance
             */
            Broker.prototype.name = "";

            /**
             * Broker uuid.
             * @member {string} uuid
             * @memberof mainvec.iunet.Broker
             * @instance
             */
            Broker.prototype.uuid = "";

            /**
             * Broker desc.
             * @member {string} desc
             * @memberof mainvec.iunet.Broker
             * @instance
             */
            Broker.prototype.desc = "";

            /**
             * Broker type.
             * @member {string} type
             * @memberof mainvec.iunet.Broker
             * @instance
             */
            Broker.prototype.type = "";

            /**
             * Broker connParams.
             * @member {string} connParams
             * @memberof mainvec.iunet.Broker
             * @instance
             */
            Broker.prototype.connParams = "";

            /**
             * Broker authParams.
             * @member {string} authParams
             * @memberof mainvec.iunet.Broker
             * @instance
             */
            Broker.prototype.authParams = "";

            /**
             * Broker disabled.
             * @member {boolean} disabled
             * @memberof mainvec.iunet.Broker
             * @instance
             */
            Broker.prototype.disabled = false;

            /**
             * Creates a new Broker instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.Broker
             * @static
             * @param {mainvec.iunet.IBroker=} [properties] Properties to set
             * @returns {mainvec.iunet.Broker} Broker instance
             */
            Broker.create = function create(properties) {
                return new Broker(properties);
            };

            /**
             * Encodes the specified Broker message. Does not implicitly {@link mainvec.iunet.Broker.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.Broker
             * @static
             * @param {mainvec.iunet.IBroker} message Broker message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Broker.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.uuid != null && Object.hasOwnProperty.call(message, "uuid"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.uuid);
                if (message.desc != null && Object.hasOwnProperty.call(message, "desc"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.desc);
                if (message.type != null && Object.hasOwnProperty.call(message, "type"))
                    writer.uint32(/* id 4, wireType 2 =*/34).string(message.type);
                if (message.connParams != null && Object.hasOwnProperty.call(message, "connParams"))
                    writer.uint32(/* id 6, wireType 2 =*/50).string(message.connParams);
                if (message.authParams != null && Object.hasOwnProperty.call(message, "authParams"))
                    writer.uint32(/* id 8, wireType 2 =*/66).string(message.authParams);
                if (message.disabled != null && Object.hasOwnProperty.call(message, "disabled"))
                    writer.uint32(/* id 10, wireType 0 =*/80).bool(message.disabled);
                return writer;
            };

            /**
             * Encodes the specified Broker message, length delimited. Does not implicitly {@link mainvec.iunet.Broker.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.Broker
             * @static
             * @param {mainvec.iunet.IBroker} message Broker message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Broker.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a Broker message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.Broker
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.Broker} Broker
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Broker.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.Broker();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.uuid = reader.string();
                            break;
                        }
                    case 3: {
                            message.desc = reader.string();
                            break;
                        }
                    case 4: {
                            message.type = reader.string();
                            break;
                        }
                    case 6: {
                            message.connParams = reader.string();
                            break;
                        }
                    case 8: {
                            message.authParams = reader.string();
                            break;
                        }
                    case 10: {
                            message.disabled = reader.bool();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a Broker message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.Broker
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.Broker} Broker
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Broker.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a Broker message.
             * @function verify
             * @memberof mainvec.iunet.Broker
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Broker.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    if (!$util.isString(message.uuid))
                        return "uuid: string expected";
                if (message.desc != null && message.hasOwnProperty("desc"))
                    if (!$util.isString(message.desc))
                        return "desc: string expected";
                if (message.type != null && message.hasOwnProperty("type"))
                    if (!$util.isString(message.type))
                        return "type: string expected";
                if (message.connParams != null && message.hasOwnProperty("connParams"))
                    if (!$util.isString(message.connParams))
                        return "connParams: string expected";
                if (message.authParams != null && message.hasOwnProperty("authParams"))
                    if (!$util.isString(message.authParams))
                        return "authParams: string expected";
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    if (typeof message.disabled !== "boolean")
                        return "disabled: boolean expected";
                return null;
            };

            /**
             * Creates a Broker message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.Broker
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.Broker} Broker
             */
            Broker.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.Broker)
                    return object;
                let message = new $root.mainvec.iunet.Broker();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.uuid != null)
                    message.uuid = String(object.uuid);
                if (object.desc != null)
                    message.desc = String(object.desc);
                if (object.type != null)
                    message.type = String(object.type);
                if (object.connParams != null)
                    message.connParams = String(object.connParams);
                if (object.authParams != null)
                    message.authParams = String(object.authParams);
                if (object.disabled != null)
                    message.disabled = Boolean(object.disabled);
                return message;
            };

            /**
             * Creates a plain object from a Broker message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.Broker
             * @static
             * @param {mainvec.iunet.Broker} message Broker
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Broker.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.uuid = "";
                    object.desc = "";
                    object.type = "";
                    object.connParams = "";
                    object.authParams = "";
                    object.disabled = false;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    object.uuid = message.uuid;
                if (message.desc != null && message.hasOwnProperty("desc"))
                    object.desc = message.desc;
                if (message.type != null && message.hasOwnProperty("type"))
                    object.type = message.type;
                if (message.connParams != null && message.hasOwnProperty("connParams"))
                    object.connParams = message.connParams;
                if (message.authParams != null && message.hasOwnProperty("authParams"))
                    object.authParams = message.authParams;
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    object.disabled = message.disabled;
                return object;
            };

            /**
             * Converts this Broker to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.Broker
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Broker.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for Broker
             * @function getTypeUrl
             * @memberof mainvec.iunet.Broker
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            Broker.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.Broker";
            };

            return Broker;
        })();

        iunet.OperationResult = (function() {

            /**
             * Properties of an OperationResult.
             * @memberof mainvec.iunet
             * @interface IOperationResult
             * @property {boolean|null} [success] OperationResult success
             * @property {string|null} [statusCode] OperationResult statusCode
             * @property {string|null} [details] OperationResult details
             * @property {Object.<string,string>|null} [additional] OperationResult additional
             */

            /**
             * Constructs a new OperationResult.
             * @memberof mainvec.iunet
             * @classdesc Represents an OperationResult.
             * @implements IOperationResult
             * @constructor
             * @param {mainvec.iunet.IOperationResult=} [properties] Properties to set
             */
            function OperationResult(properties) {
                this.additional = {};
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * OperationResult success.
             * @member {boolean} success
             * @memberof mainvec.iunet.OperationResult
             * @instance
             */
            OperationResult.prototype.success = false;

            /**
             * OperationResult statusCode.
             * @member {string} statusCode
             * @memberof mainvec.iunet.OperationResult
             * @instance
             */
            OperationResult.prototype.statusCode = "";

            /**
             * OperationResult details.
             * @member {string} details
             * @memberof mainvec.iunet.OperationResult
             * @instance
             */
            OperationResult.prototype.details = "";

            /**
             * OperationResult additional.
             * @member {Object.<string,string>} additional
             * @memberof mainvec.iunet.OperationResult
             * @instance
             */
            OperationResult.prototype.additional = $util.emptyObject;

            /**
             * Creates a new OperationResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.OperationResult
             * @static
             * @param {mainvec.iunet.IOperationResult=} [properties] Properties to set
             * @returns {mainvec.iunet.OperationResult} OperationResult instance
             */
            OperationResult.create = function create(properties) {
                return new OperationResult(properties);
            };

            /**
             * Encodes the specified OperationResult message. Does not implicitly {@link mainvec.iunet.OperationResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.OperationResult
             * @static
             * @param {mainvec.iunet.IOperationResult} message OperationResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OperationResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.success != null && Object.hasOwnProperty.call(message, "success"))
                    writer.uint32(/* id 1, wireType 0 =*/8).bool(message.success);
                if (message.statusCode != null && Object.hasOwnProperty.call(message, "statusCode"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.statusCode);
                if (message.details != null && Object.hasOwnProperty.call(message, "details"))
                    writer.uint32(/* id 5, wireType 2 =*/42).string(message.details);
                if (message.additional != null && Object.hasOwnProperty.call(message, "additional"))
                    for (let keys = Object.keys(message.additional), i = 0; i < keys.length; ++i)
                        writer.uint32(/* id 10, wireType 2 =*/82).fork().uint32(/* id 1, wireType 2 =*/10).string(keys[i]).uint32(/* id 2, wireType 2 =*/18).string(message.additional[keys[i]]).ldelim();
                return writer;
            };

            /**
             * Encodes the specified OperationResult message, length delimited. Does not implicitly {@link mainvec.iunet.OperationResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.OperationResult
             * @static
             * @param {mainvec.iunet.IOperationResult} message OperationResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OperationResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an OperationResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.OperationResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.OperationResult} OperationResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OperationResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.OperationResult(), key, value;
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.success = reader.bool();
                            break;
                        }
                    case 3: {
                            message.statusCode = reader.string();
                            break;
                        }
                    case 5: {
                            message.details = reader.string();
                            break;
                        }
                    case 10: {
                            if (message.additional === $util.emptyObject)
                                message.additional = {};
                            let end2 = reader.uint32() + reader.pos;
                            key = "";
                            value = "";
                            while (reader.pos < end2) {
                                let tag2 = reader.uint32();
                                switch (tag2 >>> 3) {
                                case 1:
                                    key = reader.string();
                                    break;
                                case 2:
                                    value = reader.string();
                                    break;
                                default:
                                    reader.skipType(tag2 & 7);
                                    break;
                                }
                            }
                            message.additional[key] = value;
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an OperationResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.OperationResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.OperationResult} OperationResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OperationResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an OperationResult message.
             * @function verify
             * @memberof mainvec.iunet.OperationResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            OperationResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.success != null && message.hasOwnProperty("success"))
                    if (typeof message.success !== "boolean")
                        return "success: boolean expected";
                if (message.statusCode != null && message.hasOwnProperty("statusCode"))
                    if (!$util.isString(message.statusCode))
                        return "statusCode: string expected";
                if (message.details != null && message.hasOwnProperty("details"))
                    if (!$util.isString(message.details))
                        return "details: string expected";
                if (message.additional != null && message.hasOwnProperty("additional")) {
                    if (!$util.isObject(message.additional))
                        return "additional: object expected";
                    let key = Object.keys(message.additional);
                    for (let i = 0; i < key.length; ++i)
                        if (!$util.isString(message.additional[key[i]]))
                            return "additional: string{k:string} expected";
                }
                return null;
            };

            /**
             * Creates an OperationResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.OperationResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.OperationResult} OperationResult
             */
            OperationResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.OperationResult)
                    return object;
                let message = new $root.mainvec.iunet.OperationResult();
                if (object.success != null)
                    message.success = Boolean(object.success);
                if (object.statusCode != null)
                    message.statusCode = String(object.statusCode);
                if (object.details != null)
                    message.details = String(object.details);
                if (object.additional) {
                    if (typeof object.additional !== "object")
                        throw TypeError(".mainvec.iunet.OperationResult.additional: object expected");
                    message.additional = {};
                    for (let keys = Object.keys(object.additional), i = 0; i < keys.length; ++i)
                        message.additional[keys[i]] = String(object.additional[keys[i]]);
                }
                return message;
            };

            /**
             * Creates a plain object from an OperationResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.OperationResult
             * @static
             * @param {mainvec.iunet.OperationResult} message OperationResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            OperationResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.objects || options.defaults)
                    object.additional = {};
                if (options.defaults) {
                    object.success = false;
                    object.statusCode = "";
                    object.details = "";
                }
                if (message.success != null && message.hasOwnProperty("success"))
                    object.success = message.success;
                if (message.statusCode != null && message.hasOwnProperty("statusCode"))
                    object.statusCode = message.statusCode;
                if (message.details != null && message.hasOwnProperty("details"))
                    object.details = message.details;
                let keys2;
                if (message.additional && (keys2 = Object.keys(message.additional)).length) {
                    object.additional = {};
                    for (let j = 0; j < keys2.length; ++j)
                        object.additional[keys2[j]] = message.additional[keys2[j]];
                }
                return object;
            };

            /**
             * Converts this OperationResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.OperationResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            OperationResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for OperationResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.OperationResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            OperationResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.OperationResult";
            };

            return OperationResult;
        })();

        iunet.BrokerCreateCmd = (function() {

            /**
             * Properties of a BrokerCreateCmd.
             * @memberof mainvec.iunet
             * @interface IBrokerCreateCmd
             * @property {string|null} [name] BrokerCreateCmd name
             * @property {string|null} [desc] BrokerCreateCmd desc
             * @property {string|null} [type] BrokerCreateCmd type
             * @property {string|null} [connParams] BrokerCreateCmd connParams
             * @property {string|null} [authParams] BrokerCreateCmd authParams
             * @property {boolean|null} [persist] BrokerCreateCmd persist
             * @property {boolean|null} [skiptest] BrokerCreateCmd skiptest
             * @property {boolean|null} [disabled] BrokerCreateCmd disabled
             */

            /**
             * Constructs a new BrokerCreateCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents a BrokerCreateCmd.
             * @implements IBrokerCreateCmd
             * @constructor
             * @param {mainvec.iunet.IBrokerCreateCmd=} [properties] Properties to set
             */
            function BrokerCreateCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * BrokerCreateCmd name.
             * @member {string} name
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @instance
             */
            BrokerCreateCmd.prototype.name = "";

            /**
             * BrokerCreateCmd desc.
             * @member {string} desc
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @instance
             */
            BrokerCreateCmd.prototype.desc = "";

            /**
             * BrokerCreateCmd type.
             * @member {string} type
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @instance
             */
            BrokerCreateCmd.prototype.type = "";

            /**
             * BrokerCreateCmd connParams.
             * @member {string} connParams
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @instance
             */
            BrokerCreateCmd.prototype.connParams = "";

            /**
             * BrokerCreateCmd authParams.
             * @member {string} authParams
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @instance
             */
            BrokerCreateCmd.prototype.authParams = "";

            /**
             * BrokerCreateCmd persist.
             * @member {boolean} persist
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @instance
             */
            BrokerCreateCmd.prototype.persist = false;

            /**
             * BrokerCreateCmd skiptest.
             * @member {boolean} skiptest
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @instance
             */
            BrokerCreateCmd.prototype.skiptest = false;

            /**
             * BrokerCreateCmd disabled.
             * @member {boolean} disabled
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @instance
             */
            BrokerCreateCmd.prototype.disabled = false;

            /**
             * Creates a new BrokerCreateCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @static
             * @param {mainvec.iunet.IBrokerCreateCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.BrokerCreateCmd} BrokerCreateCmd instance
             */
            BrokerCreateCmd.create = function create(properties) {
                return new BrokerCreateCmd(properties);
            };

            /**
             * Encodes the specified BrokerCreateCmd message. Does not implicitly {@link mainvec.iunet.BrokerCreateCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @static
             * @param {mainvec.iunet.IBrokerCreateCmd} message BrokerCreateCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            BrokerCreateCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.desc != null && Object.hasOwnProperty.call(message, "desc"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.desc);
                if (message.type != null && Object.hasOwnProperty.call(message, "type"))
                    writer.uint32(/* id 4, wireType 2 =*/34).string(message.type);
                if (message.connParams != null && Object.hasOwnProperty.call(message, "connParams"))
                    writer.uint32(/* id 6, wireType 2 =*/50).string(message.connParams);
                if (message.authParams != null && Object.hasOwnProperty.call(message, "authParams"))
                    writer.uint32(/* id 8, wireType 2 =*/66).string(message.authParams);
                if (message.persist != null && Object.hasOwnProperty.call(message, "persist"))
                    writer.uint32(/* id 10, wireType 0 =*/80).bool(message.persist);
                if (message.skiptest != null && Object.hasOwnProperty.call(message, "skiptest"))
                    writer.uint32(/* id 11, wireType 0 =*/88).bool(message.skiptest);
                if (message.disabled != null && Object.hasOwnProperty.call(message, "disabled"))
                    writer.uint32(/* id 12, wireType 0 =*/96).bool(message.disabled);
                return writer;
            };

            /**
             * Encodes the specified BrokerCreateCmd message, length delimited. Does not implicitly {@link mainvec.iunet.BrokerCreateCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @static
             * @param {mainvec.iunet.IBrokerCreateCmd} message BrokerCreateCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            BrokerCreateCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a BrokerCreateCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.BrokerCreateCmd} BrokerCreateCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            BrokerCreateCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.BrokerCreateCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 3: {
                            message.desc = reader.string();
                            break;
                        }
                    case 4: {
                            message.type = reader.string();
                            break;
                        }
                    case 6: {
                            message.connParams = reader.string();
                            break;
                        }
                    case 8: {
                            message.authParams = reader.string();
                            break;
                        }
                    case 10: {
                            message.persist = reader.bool();
                            break;
                        }
                    case 11: {
                            message.skiptest = reader.bool();
                            break;
                        }
                    case 12: {
                            message.disabled = reader.bool();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a BrokerCreateCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.BrokerCreateCmd} BrokerCreateCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            BrokerCreateCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a BrokerCreateCmd message.
             * @function verify
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            BrokerCreateCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.desc != null && message.hasOwnProperty("desc"))
                    if (!$util.isString(message.desc))
                        return "desc: string expected";
                if (message.type != null && message.hasOwnProperty("type"))
                    if (!$util.isString(message.type))
                        return "type: string expected";
                if (message.connParams != null && message.hasOwnProperty("connParams"))
                    if (!$util.isString(message.connParams))
                        return "connParams: string expected";
                if (message.authParams != null && message.hasOwnProperty("authParams"))
                    if (!$util.isString(message.authParams))
                        return "authParams: string expected";
                if (message.persist != null && message.hasOwnProperty("persist"))
                    if (typeof message.persist !== "boolean")
                        return "persist: boolean expected";
                if (message.skiptest != null && message.hasOwnProperty("skiptest"))
                    if (typeof message.skiptest !== "boolean")
                        return "skiptest: boolean expected";
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    if (typeof message.disabled !== "boolean")
                        return "disabled: boolean expected";
                return null;
            };

            /**
             * Creates a BrokerCreateCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.BrokerCreateCmd} BrokerCreateCmd
             */
            BrokerCreateCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.BrokerCreateCmd)
                    return object;
                let message = new $root.mainvec.iunet.BrokerCreateCmd();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.desc != null)
                    message.desc = String(object.desc);
                if (object.type != null)
                    message.type = String(object.type);
                if (object.connParams != null)
                    message.connParams = String(object.connParams);
                if (object.authParams != null)
                    message.authParams = String(object.authParams);
                if (object.persist != null)
                    message.persist = Boolean(object.persist);
                if (object.skiptest != null)
                    message.skiptest = Boolean(object.skiptest);
                if (object.disabled != null)
                    message.disabled = Boolean(object.disabled);
                return message;
            };

            /**
             * Creates a plain object from a BrokerCreateCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @static
             * @param {mainvec.iunet.BrokerCreateCmd} message BrokerCreateCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            BrokerCreateCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.desc = "";
                    object.type = "";
                    object.connParams = "";
                    object.authParams = "";
                    object.persist = false;
                    object.skiptest = false;
                    object.disabled = false;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.desc != null && message.hasOwnProperty("desc"))
                    object.desc = message.desc;
                if (message.type != null && message.hasOwnProperty("type"))
                    object.type = message.type;
                if (message.connParams != null && message.hasOwnProperty("connParams"))
                    object.connParams = message.connParams;
                if (message.authParams != null && message.hasOwnProperty("authParams"))
                    object.authParams = message.authParams;
                if (message.persist != null && message.hasOwnProperty("persist"))
                    object.persist = message.persist;
                if (message.skiptest != null && message.hasOwnProperty("skiptest"))
                    object.skiptest = message.skiptest;
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    object.disabled = message.disabled;
                return object;
            };

            /**
             * Converts this BrokerCreateCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            BrokerCreateCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for BrokerCreateCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.BrokerCreateCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            BrokerCreateCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.BrokerCreateCmd";
            };

            return BrokerCreateCmd;
        })();

        iunet.BrokerCreateCmdResult = (function() {

            /**
             * Properties of a BrokerCreateCmdResult.
             * @memberof mainvec.iunet
             * @interface IBrokerCreateCmdResult
             * @property {mainvec.iunet.IBroker|null} [broker] BrokerCreateCmdResult broker
             */

            /**
             * Constructs a new BrokerCreateCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents a BrokerCreateCmdResult.
             * @implements IBrokerCreateCmdResult
             * @constructor
             * @param {mainvec.iunet.IBrokerCreateCmdResult=} [properties] Properties to set
             */
            function BrokerCreateCmdResult(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * BrokerCreateCmdResult broker.
             * @member {mainvec.iunet.IBroker|null|undefined} broker
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @instance
             */
            BrokerCreateCmdResult.prototype.broker = null;

            /**
             * Creates a new BrokerCreateCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @static
             * @param {mainvec.iunet.IBrokerCreateCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.BrokerCreateCmdResult} BrokerCreateCmdResult instance
             */
            BrokerCreateCmdResult.create = function create(properties) {
                return new BrokerCreateCmdResult(properties);
            };

            /**
             * Encodes the specified BrokerCreateCmdResult message. Does not implicitly {@link mainvec.iunet.BrokerCreateCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @static
             * @param {mainvec.iunet.IBrokerCreateCmdResult} message BrokerCreateCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            BrokerCreateCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.broker != null && Object.hasOwnProperty.call(message, "broker"))
                    $root.mainvec.iunet.Broker.encode(message.broker, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified BrokerCreateCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.BrokerCreateCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @static
             * @param {mainvec.iunet.IBrokerCreateCmdResult} message BrokerCreateCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            BrokerCreateCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a BrokerCreateCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.BrokerCreateCmdResult} BrokerCreateCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            BrokerCreateCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.BrokerCreateCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.broker = $root.mainvec.iunet.Broker.decode(reader, reader.uint32());
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a BrokerCreateCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.BrokerCreateCmdResult} BrokerCreateCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            BrokerCreateCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a BrokerCreateCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            BrokerCreateCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.broker != null && message.hasOwnProperty("broker")) {
                    let error = $root.mainvec.iunet.Broker.verify(message.broker);
                    if (error)
                        return "broker." + error;
                }
                return null;
            };

            /**
             * Creates a BrokerCreateCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.BrokerCreateCmdResult} BrokerCreateCmdResult
             */
            BrokerCreateCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.BrokerCreateCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.BrokerCreateCmdResult();
                if (object.broker != null) {
                    if (typeof object.broker !== "object")
                        throw TypeError(".mainvec.iunet.BrokerCreateCmdResult.broker: object expected");
                    message.broker = $root.mainvec.iunet.Broker.fromObject(object.broker);
                }
                return message;
            };

            /**
             * Creates a plain object from a BrokerCreateCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @static
             * @param {mainvec.iunet.BrokerCreateCmdResult} message BrokerCreateCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            BrokerCreateCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults)
                    object.broker = null;
                if (message.broker != null && message.hasOwnProperty("broker"))
                    object.broker = $root.mainvec.iunet.Broker.toObject(message.broker, options);
                return object;
            };

            /**
             * Converts this BrokerCreateCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            BrokerCreateCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for BrokerCreateCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.BrokerCreateCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            BrokerCreateCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.BrokerCreateCmdResult";
            };

            return BrokerCreateCmdResult;
        })();

        iunet.BrokerListCmd = (function() {

            /**
             * Properties of a BrokerListCmd.
             * @memberof mainvec.iunet
             * @interface IBrokerListCmd
             * @property {string|null} [filter] BrokerListCmd filter
             */

            /**
             * Constructs a new BrokerListCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents a BrokerListCmd.
             * @implements IBrokerListCmd
             * @constructor
             * @param {mainvec.iunet.IBrokerListCmd=} [properties] Properties to set
             */
            function BrokerListCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * BrokerListCmd filter.
             * @member {string} filter
             * @memberof mainvec.iunet.BrokerListCmd
             * @instance
             */
            BrokerListCmd.prototype.filter = "";

            /**
             * Creates a new BrokerListCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.BrokerListCmd
             * @static
             * @param {mainvec.iunet.IBrokerListCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.BrokerListCmd} BrokerListCmd instance
             */
            BrokerListCmd.create = function create(properties) {
                return new BrokerListCmd(properties);
            };

            /**
             * Encodes the specified BrokerListCmd message. Does not implicitly {@link mainvec.iunet.BrokerListCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.BrokerListCmd
             * @static
             * @param {mainvec.iunet.IBrokerListCmd} message BrokerListCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            BrokerListCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.filter != null && Object.hasOwnProperty.call(message, "filter"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.filter);
                return writer;
            };

            /**
             * Encodes the specified BrokerListCmd message, length delimited. Does not implicitly {@link mainvec.iunet.BrokerListCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.BrokerListCmd
             * @static
             * @param {mainvec.iunet.IBrokerListCmd} message BrokerListCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            BrokerListCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a BrokerListCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.BrokerListCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.BrokerListCmd} BrokerListCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            BrokerListCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.BrokerListCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.filter = reader.string();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a BrokerListCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.BrokerListCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.BrokerListCmd} BrokerListCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            BrokerListCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a BrokerListCmd message.
             * @function verify
             * @memberof mainvec.iunet.BrokerListCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            BrokerListCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.filter != null && message.hasOwnProperty("filter"))
                    if (!$util.isString(message.filter))
                        return "filter: string expected";
                return null;
            };

            /**
             * Creates a BrokerListCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.BrokerListCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.BrokerListCmd} BrokerListCmd
             */
            BrokerListCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.BrokerListCmd)
                    return object;
                let message = new $root.mainvec.iunet.BrokerListCmd();
                if (object.filter != null)
                    message.filter = String(object.filter);
                return message;
            };

            /**
             * Creates a plain object from a BrokerListCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.BrokerListCmd
             * @static
             * @param {mainvec.iunet.BrokerListCmd} message BrokerListCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            BrokerListCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults)
                    object.filter = "";
                if (message.filter != null && message.hasOwnProperty("filter"))
                    object.filter = message.filter;
                return object;
            };

            /**
             * Converts this BrokerListCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.BrokerListCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            BrokerListCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for BrokerListCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.BrokerListCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            BrokerListCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.BrokerListCmd";
            };

            return BrokerListCmd;
        })();

        iunet.BrokerListCmdResult = (function() {

            /**
             * Properties of a BrokerListCmdResult.
             * @memberof mainvec.iunet
             * @interface IBrokerListCmdResult
             * @property {Array.<mainvec.iunet.IBroker>|null} [brokers] BrokerListCmdResult brokers
             */

            /**
             * Constructs a new BrokerListCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents a BrokerListCmdResult.
             * @implements IBrokerListCmdResult
             * @constructor
             * @param {mainvec.iunet.IBrokerListCmdResult=} [properties] Properties to set
             */
            function BrokerListCmdResult(properties) {
                this.brokers = [];
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * BrokerListCmdResult brokers.
             * @member {Array.<mainvec.iunet.IBroker>} brokers
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @instance
             */
            BrokerListCmdResult.prototype.brokers = $util.emptyArray;

            /**
             * Creates a new BrokerListCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @static
             * @param {mainvec.iunet.IBrokerListCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.BrokerListCmdResult} BrokerListCmdResult instance
             */
            BrokerListCmdResult.create = function create(properties) {
                return new BrokerListCmdResult(properties);
            };

            /**
             * Encodes the specified BrokerListCmdResult message. Does not implicitly {@link mainvec.iunet.BrokerListCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @static
             * @param {mainvec.iunet.IBrokerListCmdResult} message BrokerListCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            BrokerListCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.brokers != null && message.brokers.length)
                    for (let i = 0; i < message.brokers.length; ++i)
                        $root.mainvec.iunet.Broker.encode(message.brokers[i], writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified BrokerListCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.BrokerListCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @static
             * @param {mainvec.iunet.IBrokerListCmdResult} message BrokerListCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            BrokerListCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a BrokerListCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.BrokerListCmdResult} BrokerListCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            BrokerListCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.BrokerListCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            if (!(message.brokers && message.brokers.length))
                                message.brokers = [];
                            message.brokers.push($root.mainvec.iunet.Broker.decode(reader, reader.uint32()));
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a BrokerListCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.BrokerListCmdResult} BrokerListCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            BrokerListCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a BrokerListCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            BrokerListCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.brokers != null && message.hasOwnProperty("brokers")) {
                    if (!Array.isArray(message.brokers))
                        return "brokers: array expected";
                    for (let i = 0; i < message.brokers.length; ++i) {
                        let error = $root.mainvec.iunet.Broker.verify(message.brokers[i]);
                        if (error)
                            return "brokers." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a BrokerListCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.BrokerListCmdResult} BrokerListCmdResult
             */
            BrokerListCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.BrokerListCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.BrokerListCmdResult();
                if (object.brokers) {
                    if (!Array.isArray(object.brokers))
                        throw TypeError(".mainvec.iunet.BrokerListCmdResult.brokers: array expected");
                    message.brokers = [];
                    for (let i = 0; i < object.brokers.length; ++i) {
                        if (typeof object.brokers[i] !== "object")
                            throw TypeError(".mainvec.iunet.BrokerListCmdResult.brokers: object expected");
                        message.brokers[i] = $root.mainvec.iunet.Broker.fromObject(object.brokers[i]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a BrokerListCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @static
             * @param {mainvec.iunet.BrokerListCmdResult} message BrokerListCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            BrokerListCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.arrays || options.defaults)
                    object.brokers = [];
                if (message.brokers && message.brokers.length) {
                    object.brokers = [];
                    for (let j = 0; j < message.brokers.length; ++j)
                        object.brokers[j] = $root.mainvec.iunet.Broker.toObject(message.brokers[j], options);
                }
                return object;
            };

            /**
             * Converts this BrokerListCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            BrokerListCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for BrokerListCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.BrokerListCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            BrokerListCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.BrokerListCmdResult";
            };

            return BrokerListCmdResult;
        })();

        iunet.ClientletCreateCmd = (function() {

            /**
             * Properties of a ClientletCreateCmd.
             * @memberof mainvec.iunet
             * @interface IClientletCreateCmd
             * @property {string|null} [name] ClientletCreateCmd name
             * @property {string|null} [serverletAddress] ClientletCreateCmd serverletAddress
             * @property {string|null} [hubUUID] ClientletCreateCmd hubUUID
             * @property {string|null} [broker] ClientletCreateCmd broker
             * @property {number|null} [port] ClientletCreateCmd port
             * @property {boolean|null} [persist] ClientletCreateCmd persist
             * @property {string|null} [brokerUUID] ClientletCreateCmd brokerUUID
             * @property {boolean|null} [disabled] ClientletCreateCmd disabled
             * @property {boolean|null} [manualStartup] ClientletCreateCmd manualStartup
             * @property {string|null} [desc] ClientletCreateCmd desc
             */

            /**
             * Constructs a new ClientletCreateCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents a ClientletCreateCmd.
             * @implements IClientletCreateCmd
             * @constructor
             * @param {mainvec.iunet.IClientletCreateCmd=} [properties] Properties to set
             */
            function ClientletCreateCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ClientletCreateCmd name.
             * @member {string} name
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.name = "";

            /**
             * ClientletCreateCmd serverletAddress.
             * @member {string} serverletAddress
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.serverletAddress = "";

            /**
             * ClientletCreateCmd hubUUID.
             * @member {string} hubUUID
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.hubUUID = "";

            /**
             * ClientletCreateCmd broker.
             * @member {string} broker
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.broker = "";

            /**
             * ClientletCreateCmd port.
             * @member {number} port
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.port = 0;

            /**
             * ClientletCreateCmd persist.
             * @member {boolean} persist
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.persist = false;

            /**
             * ClientletCreateCmd brokerUUID.
             * @member {string} brokerUUID
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.brokerUUID = "";

            /**
             * ClientletCreateCmd disabled.
             * @member {boolean} disabled
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.disabled = false;

            /**
             * ClientletCreateCmd manualStartup.
             * @member {boolean} manualStartup
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.manualStartup = false;

            /**
             * ClientletCreateCmd desc.
             * @member {string} desc
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             */
            ClientletCreateCmd.prototype.desc = "";

            /**
             * Creates a new ClientletCreateCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @static
             * @param {mainvec.iunet.IClientletCreateCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.ClientletCreateCmd} ClientletCreateCmd instance
             */
            ClientletCreateCmd.create = function create(properties) {
                return new ClientletCreateCmd(properties);
            };

            /**
             * Encodes the specified ClientletCreateCmd message. Does not implicitly {@link mainvec.iunet.ClientletCreateCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @static
             * @param {mainvec.iunet.IClientletCreateCmd} message ClientletCreateCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ClientletCreateCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.serverletAddress != null && Object.hasOwnProperty.call(message, "serverletAddress"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.serverletAddress);
                if (message.hubUUID != null && Object.hasOwnProperty.call(message, "hubUUID"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.hubUUID);
                if (message.broker != null && Object.hasOwnProperty.call(message, "broker"))
                    writer.uint32(/* id 5, wireType 2 =*/42).string(message.broker);
                if (message.port != null && Object.hasOwnProperty.call(message, "port"))
                    writer.uint32(/* id 6, wireType 0 =*/48).int32(message.port);
                if (message.persist != null && Object.hasOwnProperty.call(message, "persist"))
                    writer.uint32(/* id 7, wireType 0 =*/56).bool(message.persist);
                if (message.brokerUUID != null && Object.hasOwnProperty.call(message, "brokerUUID"))
                    writer.uint32(/* id 8, wireType 2 =*/66).string(message.brokerUUID);
                if (message.disabled != null && Object.hasOwnProperty.call(message, "disabled"))
                    writer.uint32(/* id 9, wireType 0 =*/72).bool(message.disabled);
                if (message.manualStartup != null && Object.hasOwnProperty.call(message, "manualStartup"))
                    writer.uint32(/* id 10, wireType 0 =*/80).bool(message.manualStartup);
                if (message.desc != null && Object.hasOwnProperty.call(message, "desc"))
                    writer.uint32(/* id 12, wireType 2 =*/98).string(message.desc);
                return writer;
            };

            /**
             * Encodes the specified ClientletCreateCmd message, length delimited. Does not implicitly {@link mainvec.iunet.ClientletCreateCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @static
             * @param {mainvec.iunet.IClientletCreateCmd} message ClientletCreateCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ClientletCreateCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ClientletCreateCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ClientletCreateCmd} ClientletCreateCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ClientletCreateCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ClientletCreateCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.serverletAddress = reader.string();
                            break;
                        }
                    case 3: {
                            message.hubUUID = reader.string();
                            break;
                        }
                    case 5: {
                            message.broker = reader.string();
                            break;
                        }
                    case 6: {
                            message.port = reader.int32();
                            break;
                        }
                    case 7: {
                            message.persist = reader.bool();
                            break;
                        }
                    case 8: {
                            message.brokerUUID = reader.string();
                            break;
                        }
                    case 9: {
                            message.disabled = reader.bool();
                            break;
                        }
                    case 10: {
                            message.manualStartup = reader.bool();
                            break;
                        }
                    case 12: {
                            message.desc = reader.string();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ClientletCreateCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ClientletCreateCmd} ClientletCreateCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ClientletCreateCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ClientletCreateCmd message.
             * @function verify
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ClientletCreateCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.serverletAddress != null && message.hasOwnProperty("serverletAddress"))
                    if (!$util.isString(message.serverletAddress))
                        return "serverletAddress: string expected";
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    if (!$util.isString(message.hubUUID))
                        return "hubUUID: string expected";
                if (message.broker != null && message.hasOwnProperty("broker"))
                    if (!$util.isString(message.broker))
                        return "broker: string expected";
                if (message.port != null && message.hasOwnProperty("port"))
                    if (!$util.isInteger(message.port))
                        return "port: integer expected";
                if (message.persist != null && message.hasOwnProperty("persist"))
                    if (typeof message.persist !== "boolean")
                        return "persist: boolean expected";
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    if (!$util.isString(message.brokerUUID))
                        return "brokerUUID: string expected";
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    if (typeof message.disabled !== "boolean")
                        return "disabled: boolean expected";
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    if (typeof message.manualStartup !== "boolean")
                        return "manualStartup: boolean expected";
                if (message.desc != null && message.hasOwnProperty("desc"))
                    if (!$util.isString(message.desc))
                        return "desc: string expected";
                return null;
            };

            /**
             * Creates a ClientletCreateCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ClientletCreateCmd} ClientletCreateCmd
             */
            ClientletCreateCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ClientletCreateCmd)
                    return object;
                let message = new $root.mainvec.iunet.ClientletCreateCmd();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.serverletAddress != null)
                    message.serverletAddress = String(object.serverletAddress);
                if (object.hubUUID != null)
                    message.hubUUID = String(object.hubUUID);
                if (object.broker != null)
                    message.broker = String(object.broker);
                if (object.port != null)
                    message.port = object.port | 0;
                if (object.persist != null)
                    message.persist = Boolean(object.persist);
                if (object.brokerUUID != null)
                    message.brokerUUID = String(object.brokerUUID);
                if (object.disabled != null)
                    message.disabled = Boolean(object.disabled);
                if (object.manualStartup != null)
                    message.manualStartup = Boolean(object.manualStartup);
                if (object.desc != null)
                    message.desc = String(object.desc);
                return message;
            };

            /**
             * Creates a plain object from a ClientletCreateCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @static
             * @param {mainvec.iunet.ClientletCreateCmd} message ClientletCreateCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ClientletCreateCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.serverletAddress = "";
                    object.hubUUID = "";
                    object.broker = "";
                    object.port = 0;
                    object.persist = false;
                    object.brokerUUID = "";
                    object.disabled = false;
                    object.manualStartup = false;
                    object.desc = "";
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.serverletAddress != null && message.hasOwnProperty("serverletAddress"))
                    object.serverletAddress = message.serverletAddress;
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    object.hubUUID = message.hubUUID;
                if (message.broker != null && message.hasOwnProperty("broker"))
                    object.broker = message.broker;
                if (message.port != null && message.hasOwnProperty("port"))
                    object.port = message.port;
                if (message.persist != null && message.hasOwnProperty("persist"))
                    object.persist = message.persist;
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    object.brokerUUID = message.brokerUUID;
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    object.disabled = message.disabled;
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    object.manualStartup = message.manualStartup;
                if (message.desc != null && message.hasOwnProperty("desc"))
                    object.desc = message.desc;
                return object;
            };

            /**
             * Converts this ClientletCreateCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ClientletCreateCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ClientletCreateCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.ClientletCreateCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ClientletCreateCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ClientletCreateCmd";
            };

            return ClientletCreateCmd;
        })();

        iunet.ClientletCreateCmdResult = (function() {

            /**
             * Properties of a ClientletCreateCmdResult.
             * @memberof mainvec.iunet
             * @interface IClientletCreateCmdResult
             * @property {mainvec.iunet.IClientlet|null} [clientlet] ClientletCreateCmdResult clientlet
             */

            /**
             * Constructs a new ClientletCreateCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents a ClientletCreateCmdResult.
             * @implements IClientletCreateCmdResult
             * @constructor
             * @param {mainvec.iunet.IClientletCreateCmdResult=} [properties] Properties to set
             */
            function ClientletCreateCmdResult(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ClientletCreateCmdResult clientlet.
             * @member {mainvec.iunet.IClientlet|null|undefined} clientlet
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @instance
             */
            ClientletCreateCmdResult.prototype.clientlet = null;

            /**
             * Creates a new ClientletCreateCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @static
             * @param {mainvec.iunet.IClientletCreateCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.ClientletCreateCmdResult} ClientletCreateCmdResult instance
             */
            ClientletCreateCmdResult.create = function create(properties) {
                return new ClientletCreateCmdResult(properties);
            };

            /**
             * Encodes the specified ClientletCreateCmdResult message. Does not implicitly {@link mainvec.iunet.ClientletCreateCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @static
             * @param {mainvec.iunet.IClientletCreateCmdResult} message ClientletCreateCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ClientletCreateCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.clientlet != null && Object.hasOwnProperty.call(message, "clientlet"))
                    $root.mainvec.iunet.Clientlet.encode(message.clientlet, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ClientletCreateCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.ClientletCreateCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @static
             * @param {mainvec.iunet.IClientletCreateCmdResult} message ClientletCreateCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ClientletCreateCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ClientletCreateCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ClientletCreateCmdResult} ClientletCreateCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ClientletCreateCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ClientletCreateCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.clientlet = $root.mainvec.iunet.Clientlet.decode(reader, reader.uint32());
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ClientletCreateCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ClientletCreateCmdResult} ClientletCreateCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ClientletCreateCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ClientletCreateCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ClientletCreateCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.clientlet != null && message.hasOwnProperty("clientlet")) {
                    let error = $root.mainvec.iunet.Clientlet.verify(message.clientlet);
                    if (error)
                        return "clientlet." + error;
                }
                return null;
            };

            /**
             * Creates a ClientletCreateCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ClientletCreateCmdResult} ClientletCreateCmdResult
             */
            ClientletCreateCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ClientletCreateCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.ClientletCreateCmdResult();
                if (object.clientlet != null) {
                    if (typeof object.clientlet !== "object")
                        throw TypeError(".mainvec.iunet.ClientletCreateCmdResult.clientlet: object expected");
                    message.clientlet = $root.mainvec.iunet.Clientlet.fromObject(object.clientlet);
                }
                return message;
            };

            /**
             * Creates a plain object from a ClientletCreateCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @static
             * @param {mainvec.iunet.ClientletCreateCmdResult} message ClientletCreateCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ClientletCreateCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults)
                    object.clientlet = null;
                if (message.clientlet != null && message.hasOwnProperty("clientlet"))
                    object.clientlet = $root.mainvec.iunet.Clientlet.toObject(message.clientlet, options);
                return object;
            };

            /**
             * Converts this ClientletCreateCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ClientletCreateCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ClientletCreateCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.ClientletCreateCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ClientletCreateCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ClientletCreateCmdResult";
            };

            return ClientletCreateCmdResult;
        })();

        iunet.ClientletOpCmd = (function() {

            /**
             * Properties of a ClientletOpCmd.
             * @memberof mainvec.iunet
             * @interface IClientletOpCmd
             * @property {string|null} [uuid] ClientletOpCmd uuid
             * @property {string|null} [op] ClientletOpCmd op
             */

            /**
             * Constructs a new ClientletOpCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents a ClientletOpCmd.
             * @implements IClientletOpCmd
             * @constructor
             * @param {mainvec.iunet.IClientletOpCmd=} [properties] Properties to set
             */
            function ClientletOpCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ClientletOpCmd uuid.
             * @member {string} uuid
             * @memberof mainvec.iunet.ClientletOpCmd
             * @instance
             */
            ClientletOpCmd.prototype.uuid = "";

            /**
             * ClientletOpCmd op.
             * @member {string} op
             * @memberof mainvec.iunet.ClientletOpCmd
             * @instance
             */
            ClientletOpCmd.prototype.op = "";

            /**
             * Creates a new ClientletOpCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ClientletOpCmd
             * @static
             * @param {mainvec.iunet.IClientletOpCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.ClientletOpCmd} ClientletOpCmd instance
             */
            ClientletOpCmd.create = function create(properties) {
                return new ClientletOpCmd(properties);
            };

            /**
             * Encodes the specified ClientletOpCmd message. Does not implicitly {@link mainvec.iunet.ClientletOpCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ClientletOpCmd
             * @static
             * @param {mainvec.iunet.IClientletOpCmd} message ClientletOpCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ClientletOpCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.uuid != null && Object.hasOwnProperty.call(message, "uuid"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.uuid);
                if (message.op != null && Object.hasOwnProperty.call(message, "op"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.op);
                return writer;
            };

            /**
             * Encodes the specified ClientletOpCmd message, length delimited. Does not implicitly {@link mainvec.iunet.ClientletOpCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ClientletOpCmd
             * @static
             * @param {mainvec.iunet.IClientletOpCmd} message ClientletOpCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ClientletOpCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ClientletOpCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ClientletOpCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ClientletOpCmd} ClientletOpCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ClientletOpCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ClientletOpCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.uuid = reader.string();
                            break;
                        }
                    case 3: {
                            message.op = reader.string();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ClientletOpCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ClientletOpCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ClientletOpCmd} ClientletOpCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ClientletOpCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ClientletOpCmd message.
             * @function verify
             * @memberof mainvec.iunet.ClientletOpCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ClientletOpCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    if (!$util.isString(message.uuid))
                        return "uuid: string expected";
                if (message.op != null && message.hasOwnProperty("op"))
                    if (!$util.isString(message.op))
                        return "op: string expected";
                return null;
            };

            /**
             * Creates a ClientletOpCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ClientletOpCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ClientletOpCmd} ClientletOpCmd
             */
            ClientletOpCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ClientletOpCmd)
                    return object;
                let message = new $root.mainvec.iunet.ClientletOpCmd();
                if (object.uuid != null)
                    message.uuid = String(object.uuid);
                if (object.op != null)
                    message.op = String(object.op);
                return message;
            };

            /**
             * Creates a plain object from a ClientletOpCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ClientletOpCmd
             * @static
             * @param {mainvec.iunet.ClientletOpCmd} message ClientletOpCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ClientletOpCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.uuid = "";
                    object.op = "";
                }
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    object.uuid = message.uuid;
                if (message.op != null && message.hasOwnProperty("op"))
                    object.op = message.op;
                return object;
            };

            /**
             * Converts this ClientletOpCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ClientletOpCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ClientletOpCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ClientletOpCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.ClientletOpCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ClientletOpCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ClientletOpCmd";
            };

            return ClientletOpCmd;
        })();

        iunet.ClientletOpCmdResult = (function() {

            /**
             * Properties of a ClientletOpCmdResult.
             * @memberof mainvec.iunet
             * @interface IClientletOpCmdResult
             * @property {mainvec.iunet.IOperationResult|null} [result] ClientletOpCmdResult result
             */

            /**
             * Constructs a new ClientletOpCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents a ClientletOpCmdResult.
             * @implements IClientletOpCmdResult
             * @constructor
             * @param {mainvec.iunet.IClientletOpCmdResult=} [properties] Properties to set
             */
            function ClientletOpCmdResult(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ClientletOpCmdResult result.
             * @member {mainvec.iunet.IOperationResult|null|undefined} result
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @instance
             */
            ClientletOpCmdResult.prototype.result = null;

            /**
             * Creates a new ClientletOpCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @static
             * @param {mainvec.iunet.IClientletOpCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.ClientletOpCmdResult} ClientletOpCmdResult instance
             */
            ClientletOpCmdResult.create = function create(properties) {
                return new ClientletOpCmdResult(properties);
            };

            /**
             * Encodes the specified ClientletOpCmdResult message. Does not implicitly {@link mainvec.iunet.ClientletOpCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @static
             * @param {mainvec.iunet.IClientletOpCmdResult} message ClientletOpCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ClientletOpCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.result != null && Object.hasOwnProperty.call(message, "result"))
                    $root.mainvec.iunet.OperationResult.encode(message.result, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ClientletOpCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.ClientletOpCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @static
             * @param {mainvec.iunet.IClientletOpCmdResult} message ClientletOpCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ClientletOpCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ClientletOpCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ClientletOpCmdResult} ClientletOpCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ClientletOpCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ClientletOpCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.result = $root.mainvec.iunet.OperationResult.decode(reader, reader.uint32());
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ClientletOpCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ClientletOpCmdResult} ClientletOpCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ClientletOpCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ClientletOpCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ClientletOpCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.result != null && message.hasOwnProperty("result")) {
                    let error = $root.mainvec.iunet.OperationResult.verify(message.result);
                    if (error)
                        return "result." + error;
                }
                return null;
            };

            /**
             * Creates a ClientletOpCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ClientletOpCmdResult} ClientletOpCmdResult
             */
            ClientletOpCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ClientletOpCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.ClientletOpCmdResult();
                if (object.result != null) {
                    if (typeof object.result !== "object")
                        throw TypeError(".mainvec.iunet.ClientletOpCmdResult.result: object expected");
                    message.result = $root.mainvec.iunet.OperationResult.fromObject(object.result);
                }
                return message;
            };

            /**
             * Creates a plain object from a ClientletOpCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @static
             * @param {mainvec.iunet.ClientletOpCmdResult} message ClientletOpCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ClientletOpCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults)
                    object.result = null;
                if (message.result != null && message.hasOwnProperty("result"))
                    object.result = $root.mainvec.iunet.OperationResult.toObject(message.result, options);
                return object;
            };

            /**
             * Converts this ClientletOpCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ClientletOpCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ClientletOpCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.ClientletOpCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ClientletOpCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ClientletOpCmdResult";
            };

            return ClientletOpCmdResult;
        })();

        iunet.ConnectCmd = (function() {

            /**
             * Properties of a ConnectCmd.
             * @memberof mainvec.iunet
             * @interface IConnectCmd
             * @property {string|null} [name] ConnectCmd name
             * @property {string|null} [serverletAddress] ConnectCmd serverletAddress
             * @property {string|null} [hubUUID] ConnectCmd hubUUID
             * @property {string|null} [broker] ConnectCmd broker
             * @property {number|null} [port] ConnectCmd port
             * @property {string|null} [brokerUUID] ConnectCmd brokerUUID
             */

            /**
             * Constructs a new ConnectCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents a ConnectCmd.
             * @implements IConnectCmd
             * @constructor
             * @param {mainvec.iunet.IConnectCmd=} [properties] Properties to set
             */
            function ConnectCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ConnectCmd name.
             * @member {string} name
             * @memberof mainvec.iunet.ConnectCmd
             * @instance
             */
            ConnectCmd.prototype.name = "";

            /**
             * ConnectCmd serverletAddress.
             * @member {string} serverletAddress
             * @memberof mainvec.iunet.ConnectCmd
             * @instance
             */
            ConnectCmd.prototype.serverletAddress = "";

            /**
             * ConnectCmd hubUUID.
             * @member {string} hubUUID
             * @memberof mainvec.iunet.ConnectCmd
             * @instance
             */
            ConnectCmd.prototype.hubUUID = "";

            /**
             * ConnectCmd broker.
             * @member {string} broker
             * @memberof mainvec.iunet.ConnectCmd
             * @instance
             */
            ConnectCmd.prototype.broker = "";

            /**
             * ConnectCmd port.
             * @member {number} port
             * @memberof mainvec.iunet.ConnectCmd
             * @instance
             */
            ConnectCmd.prototype.port = 0;

            /**
             * ConnectCmd brokerUUID.
             * @member {string} brokerUUID
             * @memberof mainvec.iunet.ConnectCmd
             * @instance
             */
            ConnectCmd.prototype.brokerUUID = "";

            /**
             * Creates a new ConnectCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ConnectCmd
             * @static
             * @param {mainvec.iunet.IConnectCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.ConnectCmd} ConnectCmd instance
             */
            ConnectCmd.create = function create(properties) {
                return new ConnectCmd(properties);
            };

            /**
             * Encodes the specified ConnectCmd message. Does not implicitly {@link mainvec.iunet.ConnectCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ConnectCmd
             * @static
             * @param {mainvec.iunet.IConnectCmd} message ConnectCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ConnectCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.serverletAddress != null && Object.hasOwnProperty.call(message, "serverletAddress"))
                    writer.uint32(/* id 2, wireType 2 =*/18).string(message.serverletAddress);
                if (message.hubUUID != null && Object.hasOwnProperty.call(message, "hubUUID"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.hubUUID);
                if (message.broker != null && Object.hasOwnProperty.call(message, "broker"))
                    writer.uint32(/* id 5, wireType 2 =*/42).string(message.broker);
                if (message.port != null && Object.hasOwnProperty.call(message, "port"))
                    writer.uint32(/* id 6, wireType 0 =*/48).int32(message.port);
                if (message.brokerUUID != null && Object.hasOwnProperty.call(message, "brokerUUID"))
                    writer.uint32(/* id 8, wireType 2 =*/66).string(message.brokerUUID);
                return writer;
            };

            /**
             * Encodes the specified ConnectCmd message, length delimited. Does not implicitly {@link mainvec.iunet.ConnectCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ConnectCmd
             * @static
             * @param {mainvec.iunet.IConnectCmd} message ConnectCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ConnectCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ConnectCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ConnectCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ConnectCmd} ConnectCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ConnectCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ConnectCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.serverletAddress = reader.string();
                            break;
                        }
                    case 3: {
                            message.hubUUID = reader.string();
                            break;
                        }
                    case 5: {
                            message.broker = reader.string();
                            break;
                        }
                    case 6: {
                            message.port = reader.int32();
                            break;
                        }
                    case 8: {
                            message.brokerUUID = reader.string();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ConnectCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ConnectCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ConnectCmd} ConnectCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ConnectCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ConnectCmd message.
             * @function verify
             * @memberof mainvec.iunet.ConnectCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ConnectCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.serverletAddress != null && message.hasOwnProperty("serverletAddress"))
                    if (!$util.isString(message.serverletAddress))
                        return "serverletAddress: string expected";
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    if (!$util.isString(message.hubUUID))
                        return "hubUUID: string expected";
                if (message.broker != null && message.hasOwnProperty("broker"))
                    if (!$util.isString(message.broker))
                        return "broker: string expected";
                if (message.port != null && message.hasOwnProperty("port"))
                    if (!$util.isInteger(message.port))
                        return "port: integer expected";
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    if (!$util.isString(message.brokerUUID))
                        return "brokerUUID: string expected";
                return null;
            };

            /**
             * Creates a ConnectCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ConnectCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ConnectCmd} ConnectCmd
             */
            ConnectCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ConnectCmd)
                    return object;
                let message = new $root.mainvec.iunet.ConnectCmd();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.serverletAddress != null)
                    message.serverletAddress = String(object.serverletAddress);
                if (object.hubUUID != null)
                    message.hubUUID = String(object.hubUUID);
                if (object.broker != null)
                    message.broker = String(object.broker);
                if (object.port != null)
                    message.port = object.port | 0;
                if (object.brokerUUID != null)
                    message.brokerUUID = String(object.brokerUUID);
                return message;
            };

            /**
             * Creates a plain object from a ConnectCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ConnectCmd
             * @static
             * @param {mainvec.iunet.ConnectCmd} message ConnectCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ConnectCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.serverletAddress = "";
                    object.hubUUID = "";
                    object.broker = "";
                    object.port = 0;
                    object.brokerUUID = "";
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.serverletAddress != null && message.hasOwnProperty("serverletAddress"))
                    object.serverletAddress = message.serverletAddress;
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    object.hubUUID = message.hubUUID;
                if (message.broker != null && message.hasOwnProperty("broker"))
                    object.broker = message.broker;
                if (message.port != null && message.hasOwnProperty("port"))
                    object.port = message.port;
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    object.brokerUUID = message.brokerUUID;
                return object;
            };

            /**
             * Converts this ConnectCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ConnectCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ConnectCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ConnectCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.ConnectCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ConnectCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ConnectCmd";
            };

            return ConnectCmd;
        })();

        iunet.ConnectCmdResult = (function() {

            /**
             * Properties of a ConnectCmdResult.
             * @memberof mainvec.iunet
             * @interface IConnectCmdResult
             * @property {mainvec.iunet.IClientlet|null} [clientlet] ConnectCmdResult clientlet
             */

            /**
             * Constructs a new ConnectCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents a ConnectCmdResult.
             * @implements IConnectCmdResult
             * @constructor
             * @param {mainvec.iunet.IConnectCmdResult=} [properties] Properties to set
             */
            function ConnectCmdResult(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ConnectCmdResult clientlet.
             * @member {mainvec.iunet.IClientlet|null|undefined} clientlet
             * @memberof mainvec.iunet.ConnectCmdResult
             * @instance
             */
            ConnectCmdResult.prototype.clientlet = null;

            /**
             * Creates a new ConnectCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ConnectCmdResult
             * @static
             * @param {mainvec.iunet.IConnectCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.ConnectCmdResult} ConnectCmdResult instance
             */
            ConnectCmdResult.create = function create(properties) {
                return new ConnectCmdResult(properties);
            };

            /**
             * Encodes the specified ConnectCmdResult message. Does not implicitly {@link mainvec.iunet.ConnectCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ConnectCmdResult
             * @static
             * @param {mainvec.iunet.IConnectCmdResult} message ConnectCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ConnectCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.clientlet != null && Object.hasOwnProperty.call(message, "clientlet"))
                    $root.mainvec.iunet.Clientlet.encode(message.clientlet, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ConnectCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.ConnectCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ConnectCmdResult
             * @static
             * @param {mainvec.iunet.IConnectCmdResult} message ConnectCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ConnectCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ConnectCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ConnectCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ConnectCmdResult} ConnectCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ConnectCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ConnectCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.clientlet = $root.mainvec.iunet.Clientlet.decode(reader, reader.uint32());
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ConnectCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ConnectCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ConnectCmdResult} ConnectCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ConnectCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ConnectCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.ConnectCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ConnectCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.clientlet != null && message.hasOwnProperty("clientlet")) {
                    let error = $root.mainvec.iunet.Clientlet.verify(message.clientlet);
                    if (error)
                        return "clientlet." + error;
                }
                return null;
            };

            /**
             * Creates a ConnectCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ConnectCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ConnectCmdResult} ConnectCmdResult
             */
            ConnectCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ConnectCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.ConnectCmdResult();
                if (object.clientlet != null) {
                    if (typeof object.clientlet !== "object")
                        throw TypeError(".mainvec.iunet.ConnectCmdResult.clientlet: object expected");
                    message.clientlet = $root.mainvec.iunet.Clientlet.fromObject(object.clientlet);
                }
                return message;
            };

            /**
             * Creates a plain object from a ConnectCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ConnectCmdResult
             * @static
             * @param {mainvec.iunet.ConnectCmdResult} message ConnectCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ConnectCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults)
                    object.clientlet = null;
                if (message.clientlet != null && message.hasOwnProperty("clientlet"))
                    object.clientlet = $root.mainvec.iunet.Clientlet.toObject(message.clientlet, options);
                return object;
            };

            /**
             * Converts this ConnectCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ConnectCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ConnectCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ConnectCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.ConnectCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ConnectCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ConnectCmdResult";
            };

            return ConnectCmdResult;
        })();

        iunet.ExposeCmd = (function() {

            /**
             * Properties of an ExposeCmd.
             * @memberof mainvec.iunet
             * @interface IExposeCmd
             * @property {string|null} [name] ExposeCmd name
             * @property {number|null} [port] ExposeCmd port
             * @property {string|null} [broker] ExposeCmd broker
             * @property {string|null} [brokerUUID] ExposeCmd brokerUUID
             */

            /**
             * Constructs a new ExposeCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents an ExposeCmd.
             * @implements IExposeCmd
             * @constructor
             * @param {mainvec.iunet.IExposeCmd=} [properties] Properties to set
             */
            function ExposeCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ExposeCmd name.
             * @member {string} name
             * @memberof mainvec.iunet.ExposeCmd
             * @instance
             */
            ExposeCmd.prototype.name = "";

            /**
             * ExposeCmd port.
             * @member {number} port
             * @memberof mainvec.iunet.ExposeCmd
             * @instance
             */
            ExposeCmd.prototype.port = 0;

            /**
             * ExposeCmd broker.
             * @member {string} broker
             * @memberof mainvec.iunet.ExposeCmd
             * @instance
             */
            ExposeCmd.prototype.broker = "";

            /**
             * ExposeCmd brokerUUID.
             * @member {string} brokerUUID
             * @memberof mainvec.iunet.ExposeCmd
             * @instance
             */
            ExposeCmd.prototype.brokerUUID = "";

            /**
             * Creates a new ExposeCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ExposeCmd
             * @static
             * @param {mainvec.iunet.IExposeCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.ExposeCmd} ExposeCmd instance
             */
            ExposeCmd.create = function create(properties) {
                return new ExposeCmd(properties);
            };

            /**
             * Encodes the specified ExposeCmd message. Does not implicitly {@link mainvec.iunet.ExposeCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ExposeCmd
             * @static
             * @param {mainvec.iunet.IExposeCmd} message ExposeCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ExposeCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.port != null && Object.hasOwnProperty.call(message, "port"))
                    writer.uint32(/* id 2, wireType 0 =*/16).int32(message.port);
                if (message.broker != null && Object.hasOwnProperty.call(message, "broker"))
                    writer.uint32(/* id 5, wireType 2 =*/42).string(message.broker);
                if (message.brokerUUID != null && Object.hasOwnProperty.call(message, "brokerUUID"))
                    writer.uint32(/* id 8, wireType 2 =*/66).string(message.brokerUUID);
                return writer;
            };

            /**
             * Encodes the specified ExposeCmd message, length delimited. Does not implicitly {@link mainvec.iunet.ExposeCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ExposeCmd
             * @static
             * @param {mainvec.iunet.IExposeCmd} message ExposeCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ExposeCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an ExposeCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ExposeCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ExposeCmd} ExposeCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ExposeCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ExposeCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.port = reader.int32();
                            break;
                        }
                    case 5: {
                            message.broker = reader.string();
                            break;
                        }
                    case 8: {
                            message.brokerUUID = reader.string();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an ExposeCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ExposeCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ExposeCmd} ExposeCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ExposeCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an ExposeCmd message.
             * @function verify
             * @memberof mainvec.iunet.ExposeCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ExposeCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.port != null && message.hasOwnProperty("port"))
                    if (!$util.isInteger(message.port))
                        return "port: integer expected";
                if (message.broker != null && message.hasOwnProperty("broker"))
                    if (!$util.isString(message.broker))
                        return "broker: string expected";
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    if (!$util.isString(message.brokerUUID))
                        return "brokerUUID: string expected";
                return null;
            };

            /**
             * Creates an ExposeCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ExposeCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ExposeCmd} ExposeCmd
             */
            ExposeCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ExposeCmd)
                    return object;
                let message = new $root.mainvec.iunet.ExposeCmd();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.port != null)
                    message.port = object.port | 0;
                if (object.broker != null)
                    message.broker = String(object.broker);
                if (object.brokerUUID != null)
                    message.brokerUUID = String(object.brokerUUID);
                return message;
            };

            /**
             * Creates a plain object from an ExposeCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ExposeCmd
             * @static
             * @param {mainvec.iunet.ExposeCmd} message ExposeCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ExposeCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.port = 0;
                    object.broker = "";
                    object.brokerUUID = "";
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.port != null && message.hasOwnProperty("port"))
                    object.port = message.port;
                if (message.broker != null && message.hasOwnProperty("broker"))
                    object.broker = message.broker;
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    object.brokerUUID = message.brokerUUID;
                return object;
            };

            /**
             * Converts this ExposeCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ExposeCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ExposeCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ExposeCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.ExposeCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ExposeCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ExposeCmd";
            };

            return ExposeCmd;
        })();

        iunet.ExposeCmdResult = (function() {

            /**
             * Properties of an ExposeCmdResult.
             * @memberof mainvec.iunet
             * @interface IExposeCmdResult
             * @property {mainvec.iunet.IServerlet|null} [serverlet] ExposeCmdResult serverlet
             */

            /**
             * Constructs a new ExposeCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents an ExposeCmdResult.
             * @implements IExposeCmdResult
             * @constructor
             * @param {mainvec.iunet.IExposeCmdResult=} [properties] Properties to set
             */
            function ExposeCmdResult(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ExposeCmdResult serverlet.
             * @member {mainvec.iunet.IServerlet|null|undefined} serverlet
             * @memberof mainvec.iunet.ExposeCmdResult
             * @instance
             */
            ExposeCmdResult.prototype.serverlet = null;

            /**
             * Creates a new ExposeCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ExposeCmdResult
             * @static
             * @param {mainvec.iunet.IExposeCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.ExposeCmdResult} ExposeCmdResult instance
             */
            ExposeCmdResult.create = function create(properties) {
                return new ExposeCmdResult(properties);
            };

            /**
             * Encodes the specified ExposeCmdResult message. Does not implicitly {@link mainvec.iunet.ExposeCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ExposeCmdResult
             * @static
             * @param {mainvec.iunet.IExposeCmdResult} message ExposeCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ExposeCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.serverlet != null && Object.hasOwnProperty.call(message, "serverlet"))
                    $root.mainvec.iunet.Serverlet.encode(message.serverlet, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ExposeCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.ExposeCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ExposeCmdResult
             * @static
             * @param {mainvec.iunet.IExposeCmdResult} message ExposeCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ExposeCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes an ExposeCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ExposeCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ExposeCmdResult} ExposeCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ExposeCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ExposeCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.serverlet = $root.mainvec.iunet.Serverlet.decode(reader, reader.uint32());
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes an ExposeCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ExposeCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ExposeCmdResult} ExposeCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ExposeCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an ExposeCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.ExposeCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ExposeCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.serverlet != null && message.hasOwnProperty("serverlet")) {
                    let error = $root.mainvec.iunet.Serverlet.verify(message.serverlet);
                    if (error)
                        return "serverlet." + error;
                }
                return null;
            };

            /**
             * Creates an ExposeCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ExposeCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ExposeCmdResult} ExposeCmdResult
             */
            ExposeCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ExposeCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.ExposeCmdResult();
                if (object.serverlet != null) {
                    if (typeof object.serverlet !== "object")
                        throw TypeError(".mainvec.iunet.ExposeCmdResult.serverlet: object expected");
                    message.serverlet = $root.mainvec.iunet.Serverlet.fromObject(object.serverlet);
                }
                return message;
            };

            /**
             * Creates a plain object from an ExposeCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ExposeCmdResult
             * @static
             * @param {mainvec.iunet.ExposeCmdResult} message ExposeCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ExposeCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults)
                    object.serverlet = null;
                if (message.serverlet != null && message.hasOwnProperty("serverlet"))
                    object.serverlet = $root.mainvec.iunet.Serverlet.toObject(message.serverlet, options);
                return object;
            };

            /**
             * Converts this ExposeCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ExposeCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ExposeCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ExposeCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.ExposeCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ExposeCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ExposeCmdResult";
            };

            return ExposeCmdResult;
        })();

        iunet.HubCreateCmd = (function() {

            /**
             * Properties of a HubCreateCmd.
             * @memberof mainvec.iunet
             * @interface IHubCreateCmd
             * @property {string|null} [name] HubCreateCmd name
             * @property {string|null} [broker] HubCreateCmd broker
             * @property {string|null} [desc] HubCreateCmd desc
             * @property {boolean|null} [persist] HubCreateCmd persist
             * @property {string|null} [brokerUUID] HubCreateCmd brokerUUID
             * @property {boolean|null} [disabled] HubCreateCmd disabled
             * @property {boolean|null} [manualStartup] HubCreateCmd manualStartup
             */

            /**
             * Constructs a new HubCreateCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents a HubCreateCmd.
             * @implements IHubCreateCmd
             * @constructor
             * @param {mainvec.iunet.IHubCreateCmd=} [properties] Properties to set
             */
            function HubCreateCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * HubCreateCmd name.
             * @member {string} name
             * @memberof mainvec.iunet.HubCreateCmd
             * @instance
             */
            HubCreateCmd.prototype.name = "";

            /**
             * HubCreateCmd broker.
             * @member {string} broker
             * @memberof mainvec.iunet.HubCreateCmd
             * @instance
             */
            HubCreateCmd.prototype.broker = "";

            /**
             * HubCreateCmd desc.
             * @member {string} desc
             * @memberof mainvec.iunet.HubCreateCmd
             * @instance
             */
            HubCreateCmd.prototype.desc = "";

            /**
             * HubCreateCmd persist.
             * @member {boolean} persist
             * @memberof mainvec.iunet.HubCreateCmd
             * @instance
             */
            HubCreateCmd.prototype.persist = false;

            /**
             * HubCreateCmd brokerUUID.
             * @member {string} brokerUUID
             * @memberof mainvec.iunet.HubCreateCmd
             * @instance
             */
            HubCreateCmd.prototype.brokerUUID = "";

            /**
             * HubCreateCmd disabled.
             * @member {boolean} disabled
             * @memberof mainvec.iunet.HubCreateCmd
             * @instance
             */
            HubCreateCmd.prototype.disabled = false;

            /**
             * HubCreateCmd manualStartup.
             * @member {boolean} manualStartup
             * @memberof mainvec.iunet.HubCreateCmd
             * @instance
             */
            HubCreateCmd.prototype.manualStartup = false;

            /**
             * Creates a new HubCreateCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.HubCreateCmd
             * @static
             * @param {mainvec.iunet.IHubCreateCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.HubCreateCmd} HubCreateCmd instance
             */
            HubCreateCmd.create = function create(properties) {
                return new HubCreateCmd(properties);
            };

            /**
             * Encodes the specified HubCreateCmd message. Does not implicitly {@link mainvec.iunet.HubCreateCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.HubCreateCmd
             * @static
             * @param {mainvec.iunet.IHubCreateCmd} message HubCreateCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            HubCreateCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.broker != null && Object.hasOwnProperty.call(message, "broker"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.broker);
                if (message.desc != null && Object.hasOwnProperty.call(message, "desc"))
                    writer.uint32(/* id 5, wireType 2 =*/42).string(message.desc);
                if (message.persist != null && Object.hasOwnProperty.call(message, "persist"))
                    writer.uint32(/* id 7, wireType 0 =*/56).bool(message.persist);
                if (message.brokerUUID != null && Object.hasOwnProperty.call(message, "brokerUUID"))
                    writer.uint32(/* id 8, wireType 2 =*/66).string(message.brokerUUID);
                if (message.disabled != null && Object.hasOwnProperty.call(message, "disabled"))
                    writer.uint32(/* id 9, wireType 0 =*/72).bool(message.disabled);
                if (message.manualStartup != null && Object.hasOwnProperty.call(message, "manualStartup"))
                    writer.uint32(/* id 10, wireType 0 =*/80).bool(message.manualStartup);
                return writer;
            };

            /**
             * Encodes the specified HubCreateCmd message, length delimited. Does not implicitly {@link mainvec.iunet.HubCreateCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.HubCreateCmd
             * @static
             * @param {mainvec.iunet.IHubCreateCmd} message HubCreateCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            HubCreateCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a HubCreateCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.HubCreateCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.HubCreateCmd} HubCreateCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            HubCreateCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.HubCreateCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 3: {
                            message.broker = reader.string();
                            break;
                        }
                    case 5: {
                            message.desc = reader.string();
                            break;
                        }
                    case 7: {
                            message.persist = reader.bool();
                            break;
                        }
                    case 8: {
                            message.brokerUUID = reader.string();
                            break;
                        }
                    case 9: {
                            message.disabled = reader.bool();
                            break;
                        }
                    case 10: {
                            message.manualStartup = reader.bool();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a HubCreateCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.HubCreateCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.HubCreateCmd} HubCreateCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            HubCreateCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a HubCreateCmd message.
             * @function verify
             * @memberof mainvec.iunet.HubCreateCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            HubCreateCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.broker != null && message.hasOwnProperty("broker"))
                    if (!$util.isString(message.broker))
                        return "broker: string expected";
                if (message.desc != null && message.hasOwnProperty("desc"))
                    if (!$util.isString(message.desc))
                        return "desc: string expected";
                if (message.persist != null && message.hasOwnProperty("persist"))
                    if (typeof message.persist !== "boolean")
                        return "persist: boolean expected";
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    if (!$util.isString(message.brokerUUID))
                        return "brokerUUID: string expected";
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    if (typeof message.disabled !== "boolean")
                        return "disabled: boolean expected";
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    if (typeof message.manualStartup !== "boolean")
                        return "manualStartup: boolean expected";
                return null;
            };

            /**
             * Creates a HubCreateCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.HubCreateCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.HubCreateCmd} HubCreateCmd
             */
            HubCreateCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.HubCreateCmd)
                    return object;
                let message = new $root.mainvec.iunet.HubCreateCmd();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.broker != null)
                    message.broker = String(object.broker);
                if (object.desc != null)
                    message.desc = String(object.desc);
                if (object.persist != null)
                    message.persist = Boolean(object.persist);
                if (object.brokerUUID != null)
                    message.brokerUUID = String(object.brokerUUID);
                if (object.disabled != null)
                    message.disabled = Boolean(object.disabled);
                if (object.manualStartup != null)
                    message.manualStartup = Boolean(object.manualStartup);
                return message;
            };

            /**
             * Creates a plain object from a HubCreateCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.HubCreateCmd
             * @static
             * @param {mainvec.iunet.HubCreateCmd} message HubCreateCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            HubCreateCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.broker = "";
                    object.desc = "";
                    object.persist = false;
                    object.brokerUUID = "";
                    object.disabled = false;
                    object.manualStartup = false;
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.broker != null && message.hasOwnProperty("broker"))
                    object.broker = message.broker;
                if (message.desc != null && message.hasOwnProperty("desc"))
                    object.desc = message.desc;
                if (message.persist != null && message.hasOwnProperty("persist"))
                    object.persist = message.persist;
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    object.brokerUUID = message.brokerUUID;
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    object.disabled = message.disabled;
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    object.manualStartup = message.manualStartup;
                return object;
            };

            /**
             * Converts this HubCreateCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.HubCreateCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            HubCreateCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for HubCreateCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.HubCreateCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            HubCreateCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.HubCreateCmd";
            };

            return HubCreateCmd;
        })();

        iunet.HubCreateCmdResult = (function() {

            /**
             * Properties of a HubCreateCmdResult.
             * @memberof mainvec.iunet
             * @interface IHubCreateCmdResult
             * @property {mainvec.iunet.IHub|null} [hub] HubCreateCmdResult hub
             */

            /**
             * Constructs a new HubCreateCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents a HubCreateCmdResult.
             * @implements IHubCreateCmdResult
             * @constructor
             * @param {mainvec.iunet.IHubCreateCmdResult=} [properties] Properties to set
             */
            function HubCreateCmdResult(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * HubCreateCmdResult hub.
             * @member {mainvec.iunet.IHub|null|undefined} hub
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @instance
             */
            HubCreateCmdResult.prototype.hub = null;

            /**
             * Creates a new HubCreateCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @static
             * @param {mainvec.iunet.IHubCreateCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.HubCreateCmdResult} HubCreateCmdResult instance
             */
            HubCreateCmdResult.create = function create(properties) {
                return new HubCreateCmdResult(properties);
            };

            /**
             * Encodes the specified HubCreateCmdResult message. Does not implicitly {@link mainvec.iunet.HubCreateCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @static
             * @param {mainvec.iunet.IHubCreateCmdResult} message HubCreateCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            HubCreateCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.hub != null && Object.hasOwnProperty.call(message, "hub"))
                    $root.mainvec.iunet.Hub.encode(message.hub, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified HubCreateCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.HubCreateCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @static
             * @param {mainvec.iunet.IHubCreateCmdResult} message HubCreateCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            HubCreateCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a HubCreateCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.HubCreateCmdResult} HubCreateCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            HubCreateCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.HubCreateCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.hub = $root.mainvec.iunet.Hub.decode(reader, reader.uint32());
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a HubCreateCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.HubCreateCmdResult} HubCreateCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            HubCreateCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a HubCreateCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            HubCreateCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.hub != null && message.hasOwnProperty("hub")) {
                    let error = $root.mainvec.iunet.Hub.verify(message.hub);
                    if (error)
                        return "hub." + error;
                }
                return null;
            };

            /**
             * Creates a HubCreateCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.HubCreateCmdResult} HubCreateCmdResult
             */
            HubCreateCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.HubCreateCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.HubCreateCmdResult();
                if (object.hub != null) {
                    if (typeof object.hub !== "object")
                        throw TypeError(".mainvec.iunet.HubCreateCmdResult.hub: object expected");
                    message.hub = $root.mainvec.iunet.Hub.fromObject(object.hub);
                }
                return message;
            };

            /**
             * Creates a plain object from a HubCreateCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @static
             * @param {mainvec.iunet.HubCreateCmdResult} message HubCreateCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            HubCreateCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults)
                    object.hub = null;
                if (message.hub != null && message.hasOwnProperty("hub"))
                    object.hub = $root.mainvec.iunet.Hub.toObject(message.hub, options);
                return object;
            };

            /**
             * Converts this HubCreateCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            HubCreateCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for HubCreateCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.HubCreateCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            HubCreateCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.HubCreateCmdResult";
            };

            return HubCreateCmdResult;
        })();

        iunet.HubListCmd = (function() {

            /**
             * Properties of a HubListCmd.
             * @memberof mainvec.iunet
             * @interface IHubListCmd
             * @property {string|null} [filter] HubListCmd filter
             * @property {boolean|null} [includeChildren] HubListCmd includeChildren
             */

            /**
             * Constructs a new HubListCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents a HubListCmd.
             * @implements IHubListCmd
             * @constructor
             * @param {mainvec.iunet.IHubListCmd=} [properties] Properties to set
             */
            function HubListCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * HubListCmd filter.
             * @member {string} filter
             * @memberof mainvec.iunet.HubListCmd
             * @instance
             */
            HubListCmd.prototype.filter = "";

            /**
             * HubListCmd includeChildren.
             * @member {boolean} includeChildren
             * @memberof mainvec.iunet.HubListCmd
             * @instance
             */
            HubListCmd.prototype.includeChildren = false;

            /**
             * Creates a new HubListCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.HubListCmd
             * @static
             * @param {mainvec.iunet.IHubListCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.HubListCmd} HubListCmd instance
             */
            HubListCmd.create = function create(properties) {
                return new HubListCmd(properties);
            };

            /**
             * Encodes the specified HubListCmd message. Does not implicitly {@link mainvec.iunet.HubListCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.HubListCmd
             * @static
             * @param {mainvec.iunet.IHubListCmd} message HubListCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            HubListCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.filter != null && Object.hasOwnProperty.call(message, "filter"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.filter);
                if (message.includeChildren != null && Object.hasOwnProperty.call(message, "includeChildren"))
                    writer.uint32(/* id 3, wireType 0 =*/24).bool(message.includeChildren);
                return writer;
            };

            /**
             * Encodes the specified HubListCmd message, length delimited. Does not implicitly {@link mainvec.iunet.HubListCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.HubListCmd
             * @static
             * @param {mainvec.iunet.IHubListCmd} message HubListCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            HubListCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a HubListCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.HubListCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.HubListCmd} HubListCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            HubListCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.HubListCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.filter = reader.string();
                            break;
                        }
                    case 3: {
                            message.includeChildren = reader.bool();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a HubListCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.HubListCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.HubListCmd} HubListCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            HubListCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a HubListCmd message.
             * @function verify
             * @memberof mainvec.iunet.HubListCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            HubListCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.filter != null && message.hasOwnProperty("filter"))
                    if (!$util.isString(message.filter))
                        return "filter: string expected";
                if (message.includeChildren != null && message.hasOwnProperty("includeChildren"))
                    if (typeof message.includeChildren !== "boolean")
                        return "includeChildren: boolean expected";
                return null;
            };

            /**
             * Creates a HubListCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.HubListCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.HubListCmd} HubListCmd
             */
            HubListCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.HubListCmd)
                    return object;
                let message = new $root.mainvec.iunet.HubListCmd();
                if (object.filter != null)
                    message.filter = String(object.filter);
                if (object.includeChildren != null)
                    message.includeChildren = Boolean(object.includeChildren);
                return message;
            };

            /**
             * Creates a plain object from a HubListCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.HubListCmd
             * @static
             * @param {mainvec.iunet.HubListCmd} message HubListCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            HubListCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.filter = "";
                    object.includeChildren = false;
                }
                if (message.filter != null && message.hasOwnProperty("filter"))
                    object.filter = message.filter;
                if (message.includeChildren != null && message.hasOwnProperty("includeChildren"))
                    object.includeChildren = message.includeChildren;
                return object;
            };

            /**
             * Converts this HubListCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.HubListCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            HubListCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for HubListCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.HubListCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            HubListCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.HubListCmd";
            };

            return HubListCmd;
        })();

        iunet.HubListCmdResult = (function() {

            /**
             * Properties of a HubListCmdResult.
             * @memberof mainvec.iunet
             * @interface IHubListCmdResult
             * @property {Array.<mainvec.iunet.IHub>|null} [hubs] HubListCmdResult hubs
             */

            /**
             * Constructs a new HubListCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents a HubListCmdResult.
             * @implements IHubListCmdResult
             * @constructor
             * @param {mainvec.iunet.IHubListCmdResult=} [properties] Properties to set
             */
            function HubListCmdResult(properties) {
                this.hubs = [];
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * HubListCmdResult hubs.
             * @member {Array.<mainvec.iunet.IHub>} hubs
             * @memberof mainvec.iunet.HubListCmdResult
             * @instance
             */
            HubListCmdResult.prototype.hubs = $util.emptyArray;

            /**
             * Creates a new HubListCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.HubListCmdResult
             * @static
             * @param {mainvec.iunet.IHubListCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.HubListCmdResult} HubListCmdResult instance
             */
            HubListCmdResult.create = function create(properties) {
                return new HubListCmdResult(properties);
            };

            /**
             * Encodes the specified HubListCmdResult message. Does not implicitly {@link mainvec.iunet.HubListCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.HubListCmdResult
             * @static
             * @param {mainvec.iunet.IHubListCmdResult} message HubListCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            HubListCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.hubs != null && message.hubs.length)
                    for (let i = 0; i < message.hubs.length; ++i)
                        $root.mainvec.iunet.Hub.encode(message.hubs[i], writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified HubListCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.HubListCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.HubListCmdResult
             * @static
             * @param {mainvec.iunet.IHubListCmdResult} message HubListCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            HubListCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a HubListCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.HubListCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.HubListCmdResult} HubListCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            HubListCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.HubListCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            if (!(message.hubs && message.hubs.length))
                                message.hubs = [];
                            message.hubs.push($root.mainvec.iunet.Hub.decode(reader, reader.uint32()));
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a HubListCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.HubListCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.HubListCmdResult} HubListCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            HubListCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a HubListCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.HubListCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            HubListCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.hubs != null && message.hasOwnProperty("hubs")) {
                    if (!Array.isArray(message.hubs))
                        return "hubs: array expected";
                    for (let i = 0; i < message.hubs.length; ++i) {
                        let error = $root.mainvec.iunet.Hub.verify(message.hubs[i]);
                        if (error)
                            return "hubs." + error;
                    }
                }
                return null;
            };

            /**
             * Creates a HubListCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.HubListCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.HubListCmdResult} HubListCmdResult
             */
            HubListCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.HubListCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.HubListCmdResult();
                if (object.hubs) {
                    if (!Array.isArray(object.hubs))
                        throw TypeError(".mainvec.iunet.HubListCmdResult.hubs: array expected");
                    message.hubs = [];
                    for (let i = 0; i < object.hubs.length; ++i) {
                        if (typeof object.hubs[i] !== "object")
                            throw TypeError(".mainvec.iunet.HubListCmdResult.hubs: object expected");
                        message.hubs[i] = $root.mainvec.iunet.Hub.fromObject(object.hubs[i]);
                    }
                }
                return message;
            };

            /**
             * Creates a plain object from a HubListCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.HubListCmdResult
             * @static
             * @param {mainvec.iunet.HubListCmdResult} message HubListCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            HubListCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.arrays || options.defaults)
                    object.hubs = [];
                if (message.hubs && message.hubs.length) {
                    object.hubs = [];
                    for (let j = 0; j < message.hubs.length; ++j)
                        object.hubs[j] = $root.mainvec.iunet.Hub.toObject(message.hubs[j], options);
                }
                return object;
            };

            /**
             * Converts this HubListCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.HubListCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            HubListCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for HubListCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.HubListCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            HubListCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.HubListCmdResult";
            };

            return HubListCmdResult;
        })();

        iunet.ServerletCreateCmd = (function() {

            /**
             * Properties of a ServerletCreateCmd.
             * @memberof mainvec.iunet
             * @interface IServerletCreateCmd
             * @property {string|null} [name] ServerletCreateCmd name
             * @property {number|null} [port] ServerletCreateCmd port
             * @property {string|null} [hubUUID] ServerletCreateCmd hubUUID
             * @property {string|null} [broker] ServerletCreateCmd broker
             * @property {boolean|null} [persist] ServerletCreateCmd persist
             * @property {string|null} [brokerUUID] ServerletCreateCmd brokerUUID
             * @property {boolean|null} [disabled] ServerletCreateCmd disabled
             * @property {boolean|null} [manualStartup] ServerletCreateCmd manualStartup
             * @property {string|null} [desc] ServerletCreateCmd desc
             */

            /**
             * Constructs a new ServerletCreateCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents a ServerletCreateCmd.
             * @implements IServerletCreateCmd
             * @constructor
             * @param {mainvec.iunet.IServerletCreateCmd=} [properties] Properties to set
             */
            function ServerletCreateCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ServerletCreateCmd name.
             * @member {string} name
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             */
            ServerletCreateCmd.prototype.name = "";

            /**
             * ServerletCreateCmd port.
             * @member {number} port
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             */
            ServerletCreateCmd.prototype.port = 0;

            /**
             * ServerletCreateCmd hubUUID.
             * @member {string} hubUUID
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             */
            ServerletCreateCmd.prototype.hubUUID = "";

            /**
             * ServerletCreateCmd broker.
             * @member {string} broker
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             */
            ServerletCreateCmd.prototype.broker = "";

            /**
             * ServerletCreateCmd persist.
             * @member {boolean} persist
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             */
            ServerletCreateCmd.prototype.persist = false;

            /**
             * ServerletCreateCmd brokerUUID.
             * @member {string} brokerUUID
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             */
            ServerletCreateCmd.prototype.brokerUUID = "";

            /**
             * ServerletCreateCmd disabled.
             * @member {boolean} disabled
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             */
            ServerletCreateCmd.prototype.disabled = false;

            /**
             * ServerletCreateCmd manualStartup.
             * @member {boolean} manualStartup
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             */
            ServerletCreateCmd.prototype.manualStartup = false;

            /**
             * ServerletCreateCmd desc.
             * @member {string} desc
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             */
            ServerletCreateCmd.prototype.desc = "";

            /**
             * Creates a new ServerletCreateCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @static
             * @param {mainvec.iunet.IServerletCreateCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.ServerletCreateCmd} ServerletCreateCmd instance
             */
            ServerletCreateCmd.create = function create(properties) {
                return new ServerletCreateCmd(properties);
            };

            /**
             * Encodes the specified ServerletCreateCmd message. Does not implicitly {@link mainvec.iunet.ServerletCreateCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @static
             * @param {mainvec.iunet.IServerletCreateCmd} message ServerletCreateCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServerletCreateCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.name != null && Object.hasOwnProperty.call(message, "name"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.name);
                if (message.port != null && Object.hasOwnProperty.call(message, "port"))
                    writer.uint32(/* id 2, wireType 0 =*/16).int32(message.port);
                if (message.hubUUID != null && Object.hasOwnProperty.call(message, "hubUUID"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.hubUUID);
                if (message.broker != null && Object.hasOwnProperty.call(message, "broker"))
                    writer.uint32(/* id 5, wireType 2 =*/42).string(message.broker);
                if (message.persist != null && Object.hasOwnProperty.call(message, "persist"))
                    writer.uint32(/* id 7, wireType 0 =*/56).bool(message.persist);
                if (message.brokerUUID != null && Object.hasOwnProperty.call(message, "brokerUUID"))
                    writer.uint32(/* id 8, wireType 2 =*/66).string(message.brokerUUID);
                if (message.disabled != null && Object.hasOwnProperty.call(message, "disabled"))
                    writer.uint32(/* id 9, wireType 0 =*/72).bool(message.disabled);
                if (message.manualStartup != null && Object.hasOwnProperty.call(message, "manualStartup"))
                    writer.uint32(/* id 10, wireType 0 =*/80).bool(message.manualStartup);
                if (message.desc != null && Object.hasOwnProperty.call(message, "desc"))
                    writer.uint32(/* id 12, wireType 2 =*/98).string(message.desc);
                return writer;
            };

            /**
             * Encodes the specified ServerletCreateCmd message, length delimited. Does not implicitly {@link mainvec.iunet.ServerletCreateCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @static
             * @param {mainvec.iunet.IServerletCreateCmd} message ServerletCreateCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServerletCreateCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ServerletCreateCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ServerletCreateCmd} ServerletCreateCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServerletCreateCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ServerletCreateCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.name = reader.string();
                            break;
                        }
                    case 2: {
                            message.port = reader.int32();
                            break;
                        }
                    case 3: {
                            message.hubUUID = reader.string();
                            break;
                        }
                    case 5: {
                            message.broker = reader.string();
                            break;
                        }
                    case 7: {
                            message.persist = reader.bool();
                            break;
                        }
                    case 8: {
                            message.brokerUUID = reader.string();
                            break;
                        }
                    case 9: {
                            message.disabled = reader.bool();
                            break;
                        }
                    case 10: {
                            message.manualStartup = reader.bool();
                            break;
                        }
                    case 12: {
                            message.desc = reader.string();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ServerletCreateCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ServerletCreateCmd} ServerletCreateCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServerletCreateCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ServerletCreateCmd message.
             * @function verify
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ServerletCreateCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.name != null && message.hasOwnProperty("name"))
                    if (!$util.isString(message.name))
                        return "name: string expected";
                if (message.port != null && message.hasOwnProperty("port"))
                    if (!$util.isInteger(message.port))
                        return "port: integer expected";
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    if (!$util.isString(message.hubUUID))
                        return "hubUUID: string expected";
                if (message.broker != null && message.hasOwnProperty("broker"))
                    if (!$util.isString(message.broker))
                        return "broker: string expected";
                if (message.persist != null && message.hasOwnProperty("persist"))
                    if (typeof message.persist !== "boolean")
                        return "persist: boolean expected";
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    if (!$util.isString(message.brokerUUID))
                        return "brokerUUID: string expected";
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    if (typeof message.disabled !== "boolean")
                        return "disabled: boolean expected";
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    if (typeof message.manualStartup !== "boolean")
                        return "manualStartup: boolean expected";
                if (message.desc != null && message.hasOwnProperty("desc"))
                    if (!$util.isString(message.desc))
                        return "desc: string expected";
                return null;
            };

            /**
             * Creates a ServerletCreateCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ServerletCreateCmd} ServerletCreateCmd
             */
            ServerletCreateCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ServerletCreateCmd)
                    return object;
                let message = new $root.mainvec.iunet.ServerletCreateCmd();
                if (object.name != null)
                    message.name = String(object.name);
                if (object.port != null)
                    message.port = object.port | 0;
                if (object.hubUUID != null)
                    message.hubUUID = String(object.hubUUID);
                if (object.broker != null)
                    message.broker = String(object.broker);
                if (object.persist != null)
                    message.persist = Boolean(object.persist);
                if (object.brokerUUID != null)
                    message.brokerUUID = String(object.brokerUUID);
                if (object.disabled != null)
                    message.disabled = Boolean(object.disabled);
                if (object.manualStartup != null)
                    message.manualStartup = Boolean(object.manualStartup);
                if (object.desc != null)
                    message.desc = String(object.desc);
                return message;
            };

            /**
             * Creates a plain object from a ServerletCreateCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @static
             * @param {mainvec.iunet.ServerletCreateCmd} message ServerletCreateCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ServerletCreateCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.name = "";
                    object.port = 0;
                    object.hubUUID = "";
                    object.broker = "";
                    object.persist = false;
                    object.brokerUUID = "";
                    object.disabled = false;
                    object.manualStartup = false;
                    object.desc = "";
                }
                if (message.name != null && message.hasOwnProperty("name"))
                    object.name = message.name;
                if (message.port != null && message.hasOwnProperty("port"))
                    object.port = message.port;
                if (message.hubUUID != null && message.hasOwnProperty("hubUUID"))
                    object.hubUUID = message.hubUUID;
                if (message.broker != null && message.hasOwnProperty("broker"))
                    object.broker = message.broker;
                if (message.persist != null && message.hasOwnProperty("persist"))
                    object.persist = message.persist;
                if (message.brokerUUID != null && message.hasOwnProperty("brokerUUID"))
                    object.brokerUUID = message.brokerUUID;
                if (message.disabled != null && message.hasOwnProperty("disabled"))
                    object.disabled = message.disabled;
                if (message.manualStartup != null && message.hasOwnProperty("manualStartup"))
                    object.manualStartup = message.manualStartup;
                if (message.desc != null && message.hasOwnProperty("desc"))
                    object.desc = message.desc;
                return object;
            };

            /**
             * Converts this ServerletCreateCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ServerletCreateCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ServerletCreateCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.ServerletCreateCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ServerletCreateCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ServerletCreateCmd";
            };

            return ServerletCreateCmd;
        })();

        iunet.ServerletCreateCmdResult = (function() {

            /**
             * Properties of a ServerletCreateCmdResult.
             * @memberof mainvec.iunet
             * @interface IServerletCreateCmdResult
             * @property {mainvec.iunet.IServerlet|null} [serverlet] ServerletCreateCmdResult serverlet
             */

            /**
             * Constructs a new ServerletCreateCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents a ServerletCreateCmdResult.
             * @implements IServerletCreateCmdResult
             * @constructor
             * @param {mainvec.iunet.IServerletCreateCmdResult=} [properties] Properties to set
             */
            function ServerletCreateCmdResult(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ServerletCreateCmdResult serverlet.
             * @member {mainvec.iunet.IServerlet|null|undefined} serverlet
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @instance
             */
            ServerletCreateCmdResult.prototype.serverlet = null;

            /**
             * Creates a new ServerletCreateCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @static
             * @param {mainvec.iunet.IServerletCreateCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.ServerletCreateCmdResult} ServerletCreateCmdResult instance
             */
            ServerletCreateCmdResult.create = function create(properties) {
                return new ServerletCreateCmdResult(properties);
            };

            /**
             * Encodes the specified ServerletCreateCmdResult message. Does not implicitly {@link mainvec.iunet.ServerletCreateCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @static
             * @param {mainvec.iunet.IServerletCreateCmdResult} message ServerletCreateCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServerletCreateCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.serverlet != null && Object.hasOwnProperty.call(message, "serverlet"))
                    $root.mainvec.iunet.Serverlet.encode(message.serverlet, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ServerletCreateCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.ServerletCreateCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @static
             * @param {mainvec.iunet.IServerletCreateCmdResult} message ServerletCreateCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServerletCreateCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ServerletCreateCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ServerletCreateCmdResult} ServerletCreateCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServerletCreateCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ServerletCreateCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.serverlet = $root.mainvec.iunet.Serverlet.decode(reader, reader.uint32());
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ServerletCreateCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ServerletCreateCmdResult} ServerletCreateCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServerletCreateCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ServerletCreateCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ServerletCreateCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.serverlet != null && message.hasOwnProperty("serverlet")) {
                    let error = $root.mainvec.iunet.Serverlet.verify(message.serverlet);
                    if (error)
                        return "serverlet." + error;
                }
                return null;
            };

            /**
             * Creates a ServerletCreateCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ServerletCreateCmdResult} ServerletCreateCmdResult
             */
            ServerletCreateCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ServerletCreateCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.ServerletCreateCmdResult();
                if (object.serverlet != null) {
                    if (typeof object.serverlet !== "object")
                        throw TypeError(".mainvec.iunet.ServerletCreateCmdResult.serverlet: object expected");
                    message.serverlet = $root.mainvec.iunet.Serverlet.fromObject(object.serverlet);
                }
                return message;
            };

            /**
             * Creates a plain object from a ServerletCreateCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @static
             * @param {mainvec.iunet.ServerletCreateCmdResult} message ServerletCreateCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ServerletCreateCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults)
                    object.serverlet = null;
                if (message.serverlet != null && message.hasOwnProperty("serverlet"))
                    object.serverlet = $root.mainvec.iunet.Serverlet.toObject(message.serverlet, options);
                return object;
            };

            /**
             * Converts this ServerletCreateCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ServerletCreateCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ServerletCreateCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.ServerletCreateCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ServerletCreateCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ServerletCreateCmdResult";
            };

            return ServerletCreateCmdResult;
        })();

        iunet.ServerletOpCmd = (function() {

            /**
             * Properties of a ServerletOpCmd.
             * @memberof mainvec.iunet
             * @interface IServerletOpCmd
             * @property {string|null} [uuid] ServerletOpCmd uuid
             * @property {string|null} [op] ServerletOpCmd op
             */

            /**
             * Constructs a new ServerletOpCmd.
             * @memberof mainvec.iunet
             * @classdesc Represents a ServerletOpCmd.
             * @implements IServerletOpCmd
             * @constructor
             * @param {mainvec.iunet.IServerletOpCmd=} [properties] Properties to set
             */
            function ServerletOpCmd(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ServerletOpCmd uuid.
             * @member {string} uuid
             * @memberof mainvec.iunet.ServerletOpCmd
             * @instance
             */
            ServerletOpCmd.prototype.uuid = "";

            /**
             * ServerletOpCmd op.
             * @member {string} op
             * @memberof mainvec.iunet.ServerletOpCmd
             * @instance
             */
            ServerletOpCmd.prototype.op = "";

            /**
             * Creates a new ServerletOpCmd instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ServerletOpCmd
             * @static
             * @param {mainvec.iunet.IServerletOpCmd=} [properties] Properties to set
             * @returns {mainvec.iunet.ServerletOpCmd} ServerletOpCmd instance
             */
            ServerletOpCmd.create = function create(properties) {
                return new ServerletOpCmd(properties);
            };

            /**
             * Encodes the specified ServerletOpCmd message. Does not implicitly {@link mainvec.iunet.ServerletOpCmd.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ServerletOpCmd
             * @static
             * @param {mainvec.iunet.IServerletOpCmd} message ServerletOpCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServerletOpCmd.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.uuid != null && Object.hasOwnProperty.call(message, "uuid"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.uuid);
                if (message.op != null && Object.hasOwnProperty.call(message, "op"))
                    writer.uint32(/* id 3, wireType 2 =*/26).string(message.op);
                return writer;
            };

            /**
             * Encodes the specified ServerletOpCmd message, length delimited. Does not implicitly {@link mainvec.iunet.ServerletOpCmd.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ServerletOpCmd
             * @static
             * @param {mainvec.iunet.IServerletOpCmd} message ServerletOpCmd message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServerletOpCmd.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ServerletOpCmd message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ServerletOpCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ServerletOpCmd} ServerletOpCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServerletOpCmd.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ServerletOpCmd();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.uuid = reader.string();
                            break;
                        }
                    case 3: {
                            message.op = reader.string();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ServerletOpCmd message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ServerletOpCmd
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ServerletOpCmd} ServerletOpCmd
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServerletOpCmd.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ServerletOpCmd message.
             * @function verify
             * @memberof mainvec.iunet.ServerletOpCmd
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ServerletOpCmd.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    if (!$util.isString(message.uuid))
                        return "uuid: string expected";
                if (message.op != null && message.hasOwnProperty("op"))
                    if (!$util.isString(message.op))
                        return "op: string expected";
                return null;
            };

            /**
             * Creates a ServerletOpCmd message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ServerletOpCmd
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ServerletOpCmd} ServerletOpCmd
             */
            ServerletOpCmd.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ServerletOpCmd)
                    return object;
                let message = new $root.mainvec.iunet.ServerletOpCmd();
                if (object.uuid != null)
                    message.uuid = String(object.uuid);
                if (object.op != null)
                    message.op = String(object.op);
                return message;
            };

            /**
             * Creates a plain object from a ServerletOpCmd message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ServerletOpCmd
             * @static
             * @param {mainvec.iunet.ServerletOpCmd} message ServerletOpCmd
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ServerletOpCmd.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    object.uuid = "";
                    object.op = "";
                }
                if (message.uuid != null && message.hasOwnProperty("uuid"))
                    object.uuid = message.uuid;
                if (message.op != null && message.hasOwnProperty("op"))
                    object.op = message.op;
                return object;
            };

            /**
             * Converts this ServerletOpCmd to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ServerletOpCmd
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ServerletOpCmd.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ServerletOpCmd
             * @function getTypeUrl
             * @memberof mainvec.iunet.ServerletOpCmd
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ServerletOpCmd.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ServerletOpCmd";
            };

            return ServerletOpCmd;
        })();

        iunet.ServerletOpCmdResult = (function() {

            /**
             * Properties of a ServerletOpCmdResult.
             * @memberof mainvec.iunet
             * @interface IServerletOpCmdResult
             * @property {mainvec.iunet.IOperationResult|null} [result] ServerletOpCmdResult result
             */

            /**
             * Constructs a new ServerletOpCmdResult.
             * @memberof mainvec.iunet
             * @classdesc Represents a ServerletOpCmdResult.
             * @implements IServerletOpCmdResult
             * @constructor
             * @param {mainvec.iunet.IServerletOpCmdResult=} [properties] Properties to set
             */
            function ServerletOpCmdResult(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * ServerletOpCmdResult result.
             * @member {mainvec.iunet.IOperationResult|null|undefined} result
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @instance
             */
            ServerletOpCmdResult.prototype.result = null;

            /**
             * Creates a new ServerletOpCmdResult instance using the specified properties.
             * @function create
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @static
             * @param {mainvec.iunet.IServerletOpCmdResult=} [properties] Properties to set
             * @returns {mainvec.iunet.ServerletOpCmdResult} ServerletOpCmdResult instance
             */
            ServerletOpCmdResult.create = function create(properties) {
                return new ServerletOpCmdResult(properties);
            };

            /**
             * Encodes the specified ServerletOpCmdResult message. Does not implicitly {@link mainvec.iunet.ServerletOpCmdResult.verify|verify} messages.
             * @function encode
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @static
             * @param {mainvec.iunet.IServerletOpCmdResult} message ServerletOpCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServerletOpCmdResult.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.result != null && Object.hasOwnProperty.call(message, "result"))
                    $root.mainvec.iunet.OperationResult.encode(message.result, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };

            /**
             * Encodes the specified ServerletOpCmdResult message, length delimited. Does not implicitly {@link mainvec.iunet.ServerletOpCmdResult.verify|verify} messages.
             * @function encodeDelimited
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @static
             * @param {mainvec.iunet.IServerletOpCmdResult} message ServerletOpCmdResult message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServerletOpCmdResult.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a ServerletOpCmdResult message from the specified reader or buffer.
             * @function decode
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {mainvec.iunet.ServerletOpCmdResult} ServerletOpCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServerletOpCmdResult.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.mainvec.iunet.ServerletOpCmdResult();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.result = $root.mainvec.iunet.OperationResult.decode(reader, reader.uint32());
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a ServerletOpCmdResult message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {mainvec.iunet.ServerletOpCmdResult} ServerletOpCmdResult
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServerletOpCmdResult.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a ServerletOpCmdResult message.
             * @function verify
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ServerletOpCmdResult.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.result != null && message.hasOwnProperty("result")) {
                    let error = $root.mainvec.iunet.OperationResult.verify(message.result);
                    if (error)
                        return "result." + error;
                }
                return null;
            };

            /**
             * Creates a ServerletOpCmdResult message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {mainvec.iunet.ServerletOpCmdResult} ServerletOpCmdResult
             */
            ServerletOpCmdResult.fromObject = function fromObject(object) {
                if (object instanceof $root.mainvec.iunet.ServerletOpCmdResult)
                    return object;
                let message = new $root.mainvec.iunet.ServerletOpCmdResult();
                if (object.result != null) {
                    if (typeof object.result !== "object")
                        throw TypeError(".mainvec.iunet.ServerletOpCmdResult.result: object expected");
                    message.result = $root.mainvec.iunet.OperationResult.fromObject(object.result);
                }
                return message;
            };

            /**
             * Creates a plain object from a ServerletOpCmdResult message. Also converts values to other types if specified.
             * @function toObject
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @static
             * @param {mainvec.iunet.ServerletOpCmdResult} message ServerletOpCmdResult
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ServerletOpCmdResult.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults)
                    object.result = null;
                if (message.result != null && message.hasOwnProperty("result"))
                    object.result = $root.mainvec.iunet.OperationResult.toObject(message.result, options);
                return object;
            };

            /**
             * Converts this ServerletOpCmdResult to JSON.
             * @function toJSON
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ServerletOpCmdResult.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for ServerletOpCmdResult
             * @function getTypeUrl
             * @memberof mainvec.iunet.ServerletOpCmdResult
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            ServerletOpCmdResult.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/mainvec.iunet.ServerletOpCmdResult";
            };

            return ServerletOpCmdResult;
        })();

        return iunet;
    })();

    return mainvec;
})();

export const google = $root.google = (() => {

    /**
     * Namespace google.
     * @exports google
     * @namespace
     */
    const google = {};

    google.protobuf = (function() {

        /**
         * Namespace protobuf.
         * @memberof google
         * @namespace
         */
        const protobuf = {};

        protobuf.Duration = (function() {

            /**
             * Properties of a Duration.
             * @memberof google.protobuf
             * @interface IDuration
             * @property {number|Long|null} [seconds] Duration seconds
             * @property {number|null} [nanos] Duration nanos
             */

            /**
             * Constructs a new Duration.
             * @memberof google.protobuf
             * @classdesc Represents a Duration.
             * @implements IDuration
             * @constructor
             * @param {google.protobuf.IDuration=} [properties] Properties to set
             */
            function Duration(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Duration seconds.
             * @member {number|Long} seconds
             * @memberof google.protobuf.Duration
             * @instance
             */
            Duration.prototype.seconds = $util.Long ? $util.Long.fromBits(0,0,false) : 0;

            /**
             * Duration nanos.
             * @member {number} nanos
             * @memberof google.protobuf.Duration
             * @instance
             */
            Duration.prototype.nanos = 0;

            /**
             * Creates a new Duration instance using the specified properties.
             * @function create
             * @memberof google.protobuf.Duration
             * @static
             * @param {google.protobuf.IDuration=} [properties] Properties to set
             * @returns {google.protobuf.Duration} Duration instance
             */
            Duration.create = function create(properties) {
                return new Duration(properties);
            };

            /**
             * Encodes the specified Duration message. Does not implicitly {@link google.protobuf.Duration.verify|verify} messages.
             * @function encode
             * @memberof google.protobuf.Duration
             * @static
             * @param {google.protobuf.IDuration} message Duration message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Duration.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.seconds != null && Object.hasOwnProperty.call(message, "seconds"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.seconds);
                if (message.nanos != null && Object.hasOwnProperty.call(message, "nanos"))
                    writer.uint32(/* id 2, wireType 0 =*/16).int32(message.nanos);
                return writer;
            };

            /**
             * Encodes the specified Duration message, length delimited. Does not implicitly {@link google.protobuf.Duration.verify|verify} messages.
             * @function encodeDelimited
             * @memberof google.protobuf.Duration
             * @static
             * @param {google.protobuf.IDuration} message Duration message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Duration.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a Duration message from the specified reader or buffer.
             * @function decode
             * @memberof google.protobuf.Duration
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {google.protobuf.Duration} Duration
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Duration.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.google.protobuf.Duration();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.seconds = reader.int64();
                            break;
                        }
                    case 2: {
                            message.nanos = reader.int32();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a Duration message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof google.protobuf.Duration
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {google.protobuf.Duration} Duration
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Duration.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a Duration message.
             * @function verify
             * @memberof google.protobuf.Duration
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Duration.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.seconds != null && message.hasOwnProperty("seconds"))
                    if (!$util.isInteger(message.seconds) && !(message.seconds && $util.isInteger(message.seconds.low) && $util.isInteger(message.seconds.high)))
                        return "seconds: integer|Long expected";
                if (message.nanos != null && message.hasOwnProperty("nanos"))
                    if (!$util.isInteger(message.nanos))
                        return "nanos: integer expected";
                return null;
            };

            /**
             * Creates a Duration message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof google.protobuf.Duration
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {google.protobuf.Duration} Duration
             */
            Duration.fromObject = function fromObject(object) {
                if (object instanceof $root.google.protobuf.Duration)
                    return object;
                let message = new $root.google.protobuf.Duration();
                if (object.seconds != null)
                    if ($util.Long)
                        (message.seconds = $util.Long.fromValue(object.seconds)).unsigned = false;
                    else if (typeof object.seconds === "string")
                        message.seconds = parseInt(object.seconds, 10);
                    else if (typeof object.seconds === "number")
                        message.seconds = object.seconds;
                    else if (typeof object.seconds === "object")
                        message.seconds = new $util.LongBits(object.seconds.low >>> 0, object.seconds.high >>> 0).toNumber();
                if (object.nanos != null)
                    message.nanos = object.nanos | 0;
                return message;
            };

            /**
             * Creates a plain object from a Duration message. Also converts values to other types if specified.
             * @function toObject
             * @memberof google.protobuf.Duration
             * @static
             * @param {google.protobuf.Duration} message Duration
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Duration.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    if ($util.Long) {
                        let long = new $util.Long(0, 0, false);
                        object.seconds = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.seconds = options.longs === String ? "0" : 0;
                    object.nanos = 0;
                }
                if (message.seconds != null && message.hasOwnProperty("seconds"))
                    if (typeof message.seconds === "number")
                        object.seconds = options.longs === String ? String(message.seconds) : message.seconds;
                    else
                        object.seconds = options.longs === String ? $util.Long.prototype.toString.call(message.seconds) : options.longs === Number ? new $util.LongBits(message.seconds.low >>> 0, message.seconds.high >>> 0).toNumber() : message.seconds;
                if (message.nanos != null && message.hasOwnProperty("nanos"))
                    object.nanos = message.nanos;
                return object;
            };

            /**
             * Converts this Duration to JSON.
             * @function toJSON
             * @memberof google.protobuf.Duration
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Duration.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for Duration
             * @function getTypeUrl
             * @memberof google.protobuf.Duration
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            Duration.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/google.protobuf.Duration";
            };

            return Duration;
        })();

        protobuf.Timestamp = (function() {

            /**
             * Properties of a Timestamp.
             * @memberof google.protobuf
             * @interface ITimestamp
             * @property {number|Long|null} [seconds] Timestamp seconds
             * @property {number|null} [nanos] Timestamp nanos
             */

            /**
             * Constructs a new Timestamp.
             * @memberof google.protobuf
             * @classdesc Represents a Timestamp.
             * @implements ITimestamp
             * @constructor
             * @param {google.protobuf.ITimestamp=} [properties] Properties to set
             */
            function Timestamp(properties) {
                if (properties)
                    for (let keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }

            /**
             * Timestamp seconds.
             * @member {number|Long} seconds
             * @memberof google.protobuf.Timestamp
             * @instance
             */
            Timestamp.prototype.seconds = $util.Long ? $util.Long.fromBits(0,0,false) : 0;

            /**
             * Timestamp nanos.
             * @member {number} nanos
             * @memberof google.protobuf.Timestamp
             * @instance
             */
            Timestamp.prototype.nanos = 0;

            /**
             * Creates a new Timestamp instance using the specified properties.
             * @function create
             * @memberof google.protobuf.Timestamp
             * @static
             * @param {google.protobuf.ITimestamp=} [properties] Properties to set
             * @returns {google.protobuf.Timestamp} Timestamp instance
             */
            Timestamp.create = function create(properties) {
                return new Timestamp(properties);
            };

            /**
             * Encodes the specified Timestamp message. Does not implicitly {@link google.protobuf.Timestamp.verify|verify} messages.
             * @function encode
             * @memberof google.protobuf.Timestamp
             * @static
             * @param {google.protobuf.ITimestamp} message Timestamp message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Timestamp.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.seconds != null && Object.hasOwnProperty.call(message, "seconds"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.seconds);
                if (message.nanos != null && Object.hasOwnProperty.call(message, "nanos"))
                    writer.uint32(/* id 2, wireType 0 =*/16).int32(message.nanos);
                return writer;
            };

            /**
             * Encodes the specified Timestamp message, length delimited. Does not implicitly {@link google.protobuf.Timestamp.verify|verify} messages.
             * @function encodeDelimited
             * @memberof google.protobuf.Timestamp
             * @static
             * @param {google.protobuf.ITimestamp} message Timestamp message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Timestamp.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };

            /**
             * Decodes a Timestamp message from the specified reader or buffer.
             * @function decode
             * @memberof google.protobuf.Timestamp
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {google.protobuf.Timestamp} Timestamp
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Timestamp.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                let end = length === undefined ? reader.len : reader.pos + length, message = new $root.google.protobuf.Timestamp();
                while (reader.pos < end) {
                    let tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1: {
                            message.seconds = reader.int64();
                            break;
                        }
                    case 2: {
                            message.nanos = reader.int32();
                            break;
                        }
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };

            /**
             * Decodes a Timestamp message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof google.protobuf.Timestamp
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {google.protobuf.Timestamp} Timestamp
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Timestamp.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies a Timestamp message.
             * @function verify
             * @memberof google.protobuf.Timestamp
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Timestamp.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.seconds != null && message.hasOwnProperty("seconds"))
                    if (!$util.isInteger(message.seconds) && !(message.seconds && $util.isInteger(message.seconds.low) && $util.isInteger(message.seconds.high)))
                        return "seconds: integer|Long expected";
                if (message.nanos != null && message.hasOwnProperty("nanos"))
                    if (!$util.isInteger(message.nanos))
                        return "nanos: integer expected";
                return null;
            };

            /**
             * Creates a Timestamp message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof google.protobuf.Timestamp
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {google.protobuf.Timestamp} Timestamp
             */
            Timestamp.fromObject = function fromObject(object) {
                if (object instanceof $root.google.protobuf.Timestamp)
                    return object;
                let message = new $root.google.protobuf.Timestamp();
                if (object.seconds != null)
                    if ($util.Long)
                        (message.seconds = $util.Long.fromValue(object.seconds)).unsigned = false;
                    else if (typeof object.seconds === "string")
                        message.seconds = parseInt(object.seconds, 10);
                    else if (typeof object.seconds === "number")
                        message.seconds = object.seconds;
                    else if (typeof object.seconds === "object")
                        message.seconds = new $util.LongBits(object.seconds.low >>> 0, object.seconds.high >>> 0).toNumber();
                if (object.nanos != null)
                    message.nanos = object.nanos | 0;
                return message;
            };

            /**
             * Creates a plain object from a Timestamp message. Also converts values to other types if specified.
             * @function toObject
             * @memberof google.protobuf.Timestamp
             * @static
             * @param {google.protobuf.Timestamp} message Timestamp
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Timestamp.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                let object = {};
                if (options.defaults) {
                    if ($util.Long) {
                        let long = new $util.Long(0, 0, false);
                        object.seconds = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.seconds = options.longs === String ? "0" : 0;
                    object.nanos = 0;
                }
                if (message.seconds != null && message.hasOwnProperty("seconds"))
                    if (typeof message.seconds === "number")
                        object.seconds = options.longs === String ? String(message.seconds) : message.seconds;
                    else
                        object.seconds = options.longs === String ? $util.Long.prototype.toString.call(message.seconds) : options.longs === Number ? new $util.LongBits(message.seconds.low >>> 0, message.seconds.high >>> 0).toNumber() : message.seconds;
                if (message.nanos != null && message.hasOwnProperty("nanos"))
                    object.nanos = message.nanos;
                return object;
            };

            /**
             * Converts this Timestamp to JSON.
             * @function toJSON
             * @memberof google.protobuf.Timestamp
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Timestamp.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the default type url for Timestamp
             * @function getTypeUrl
             * @memberof google.protobuf.Timestamp
             * @static
             * @param {string} [typeUrlPrefix] your custom typeUrlPrefix(default "type.googleapis.com")
             * @returns {string} The default type url
             */
            Timestamp.getTypeUrl = function getTypeUrl(typeUrlPrefix) {
                if (typeUrlPrefix === undefined) {
                    typeUrlPrefix = "type.googleapis.com";
                }
                return typeUrlPrefix + "/google.protobuf.Timestamp";
            };

            return Timestamp;
        })();

        return protobuf;
    })();

    return google;
})();

export { $root as default };
