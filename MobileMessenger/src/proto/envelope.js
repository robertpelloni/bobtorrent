/*eslint-disable block-scoped-var, id-length, no-control-regex, no-magic-numbers, no-mixed-operators, no-prototype-builtins, no-redeclare, no-shadow, no-var, sort-vars, default-case, jsdoc/require-param*/
"use strict";

var $protobuf = require("protobufjs/minimal");

// Common aliases
var $Reader = $protobuf.Reader, $Writer = $protobuf.Writer, $util = $protobuf.util;
var $Object = $util.global.Object, $undefined = $util.global.undefined, $Error = $util.global.Error, $TypeError = $util.global.TypeError, $Number = $util.global.Number, $parseInt = $util.global.parseInt, $String = $util.global.String, $Array = $util.global.Array, $BigInt = $util.global.BigInt;

// Exported root namespace
var $root = $protobuf.roots["default"] || ($protobuf.roots["default"] = {});

$root.omnimesh = (function() {

    /**
     * Namespace omnimesh.
     * @exports omnimesh
     * @namespace
     */
    var omnimesh = {};

    omnimesh.envelope = (function() {

        /**
         * Namespace envelope.
         * @memberof omnimesh
         * @namespace
         */
        var envelope = {};

        envelope.Envelope = (function() {

            /**
             * Properties of an Envelope.
             * @typedef {Object} omnimesh.envelope.Envelope.$Properties
             * @property {Uint8Array|null} [id] Envelope id
             * @property {Uint8Array|null} [senderPubkey] Envelope senderPubkey
             * @property {number|Long|null} [timestamp] Envelope timestamp
             * @property {Uint8Array|null} [signature] Envelope signature
             * @property {omnimesh.envelope.Envelope.Type|null} [payloadType] Envelope payloadType
             * @property {Uint8Array|null} [encryptedBody] Envelope encryptedBody
             * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
             */

            /**
             * Properties of an Envelope.
             * @memberof omnimesh.envelope
             * @interface IEnvelope
             * @augments omnimesh.envelope.Envelope.$Properties
             * @deprecated Use omnimesh.envelope.Envelope.$Properties instead.
             */

            /**
             * Shape of an Envelope.
             * @typedef {omnimesh.envelope.Envelope.$Properties} omnimesh.envelope.Envelope.$Shape
             */

            /**
             * Constructs a new Envelope.
             * @memberof omnimesh.envelope
             * @classdesc Represents an Envelope.
             * @constructor
             * @param {omnimesh.envelope.Envelope.$Properties=} [properties] Properties to set
             * @property {Array.<Uint8Array>} [$unknowns] Unknown fields preserved while decoding when enabled
             */
            var Envelope = function (properties) {
                if (properties)
                    for (var keys = $Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null && keys[i] !== "__proto__")
                            this[keys[i]] = properties[keys[i]];
            };

            /**
             * Envelope id.
             * @member {Uint8Array} id
             * @memberof omnimesh.envelope.Envelope
             * @instance
             */
            Envelope.prototype.id = $util.newBuffer([]);

            /**
             * Envelope senderPubkey.
             * @member {Uint8Array} senderPubkey
             * @memberof omnimesh.envelope.Envelope
             * @instance
             */
            Envelope.prototype.senderPubkey = $util.newBuffer([]);

            /**
             * Envelope timestamp.
             * @member {number|Long} timestamp
             * @memberof omnimesh.envelope.Envelope
             * @instance
             */
            Envelope.prototype.timestamp = $util.Long ? $util.Long.fromBits(0,0,false) : 0;

            /**
             * Envelope signature.
             * @member {Uint8Array} signature
             * @memberof omnimesh.envelope.Envelope
             * @instance
             */
            Envelope.prototype.signature = $util.newBuffer([]);

            /**
             * Envelope payloadType.
             * @member {omnimesh.envelope.Envelope.Type} payloadType
             * @memberof omnimesh.envelope.Envelope
             * @instance
             */
            Envelope.prototype.payloadType = 0;

            /**
             * Envelope encryptedBody.
             * @member {Uint8Array} encryptedBody
             * @memberof omnimesh.envelope.Envelope
             * @instance
             */
            Envelope.prototype.encryptedBody = $util.newBuffer([]);

            /**
             * Creates a new Envelope instance using the specified properties.
             * @function create
             * @memberof omnimesh.envelope.Envelope
             * @static
             * @param {omnimesh.envelope.Envelope.$Properties=} [properties] Properties to set
             * @returns {omnimesh.envelope.Envelope} Envelope instance
             * @type {{
             *   (properties: omnimesh.envelope.Envelope.$Shape): omnimesh.envelope.Envelope & omnimesh.envelope.Envelope.$Shape;
             *   (properties?: omnimesh.envelope.Envelope.$Properties): omnimesh.envelope.Envelope;
             * }}
             */
            Envelope.create = function(properties) {
                return new Envelope(properties);
            };

            /**
             * Encodes the specified Envelope message. Does not implicitly {@link omnimesh.envelope.Envelope.verify|verify} messages.
             * @function encode
             * @memberof omnimesh.envelope.Envelope
             * @static
             * @param {omnimesh.envelope.Envelope.$Properties} message Envelope message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Envelope.encode = function (message, writer, _depth) {
                if (!writer)
                    writer = $Writer.create();
                if (_depth === $undefined)
                    _depth = 0;
                if (_depth > $util.recursionLimit)
                    throw $Error("max depth exceeded");
                if (message.id != null && $Object.hasOwnProperty.call(message, "id"))
                    writer.uint32(/* id 1, wireType 2 =*/10).bytes(message.id);
                if (message.senderPubkey != null && $Object.hasOwnProperty.call(message, "senderPubkey"))
                    writer.uint32(/* id 2, wireType 2 =*/18).bytes(message.senderPubkey);
                if (message.timestamp != null && $Object.hasOwnProperty.call(message, "timestamp"))
                    writer.uint32(/* id 3, wireType 0 =*/24).int64(message.timestamp);
                if (message.signature != null && $Object.hasOwnProperty.call(message, "signature"))
                    writer.uint32(/* id 4, wireType 2 =*/34).bytes(message.signature);
                if (message.payloadType != null && $Object.hasOwnProperty.call(message, "payloadType"))
                    writer.uint32(/* id 5, wireType 0 =*/40).int32(message.payloadType);
                if (message.encryptedBody != null && $Object.hasOwnProperty.call(message, "encryptedBody"))
                    writer.uint32(/* id 6, wireType 2 =*/50).bytes(message.encryptedBody);
                if (message.$unknowns != null && $Object.hasOwnProperty.call(message, "$unknowns"))
                    for (var i = 0; i < message.$unknowns.length; ++i)
                        writer.raw(message.$unknowns[i]);
                return writer;
            };

            /**
             * Encodes the specified Envelope message, length delimited. Does not implicitly {@link omnimesh.envelope.Envelope.verify|verify} messages.
             * @function encodeDelimited
             * @memberof omnimesh.envelope.Envelope
             * @static
             * @param {omnimesh.envelope.Envelope.$Properties} message Envelope message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Envelope.encodeDelimited = function(message, writer) {
                return this.encode(message, writer && writer.len ? writer.fork() : writer).ldelim();
            };

            /**
             * Decodes an Envelope message from the specified reader or buffer.
             * @function decode
             * @memberof omnimesh.envelope.Envelope
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {omnimesh.envelope.Envelope & omnimesh.envelope.Envelope.$Shape} Envelope
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Envelope.decode = function (reader, length, _end, _depth, _target) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                if (_depth === $undefined)
                    _depth = 0;
                if (_depth > $Reader.recursionLimit)
                    throw $Error("max depth exceeded");
                var end = length === $undefined ? reader.len : reader.pos + length, message = _target || new $root.omnimesh.envelope.Envelope(), value;
                while (reader.pos < end) {
                    var start = reader.pos;
                    var tag = reader.tag();
                    if (tag === _end) {
                        _end = $undefined;
                        break;
                    }
                    var wireType = tag & 7;
                    switch (tag >>>= 3) {
                    case 1: {
                            if (wireType !== 2)
                                break;
                            if ((value = reader.bytes()).length)
                                message.id = value;
                            else
                                delete message.id;
                            continue;
                        }
                    case 2: {
                            if (wireType !== 2)
                                break;
                            if ((value = reader.bytes()).length)
                                message.senderPubkey = value;
                            else
                                delete message.senderPubkey;
                            continue;
                        }
                    case 3: {
                            if (wireType !== 0)
                                break;
                            if (typeof (value = reader.int64()) === "object" ? value.low || value.high : value !== 0)
                                message.timestamp = value;
                            else
                                delete message.timestamp;
                            continue;
                        }
                    case 4: {
                            if (wireType !== 2)
                                break;
                            if ((value = reader.bytes()).length)
                                message.signature = value;
                            else
                                delete message.signature;
                            continue;
                        }
                    case 5: {
                            if (wireType !== 0)
                                break;
                            if (value = reader.int32())
                                message.payloadType = value;
                            else
                                delete message.payloadType;
                            continue;
                        }
                    case 6: {
                            if (wireType !== 2)
                                break;
                            if ((value = reader.bytes()).length)
                                message.encryptedBody = value;
                            else
                                delete message.encryptedBody;
                            continue;
                        }
                    }
                    reader.skipType(wireType, _depth, tag);
                    if (!reader.discardUnknown) {
                        $util.makeProp(message, "$unknowns", false);
                        (message.$unknowns || (message.$unknowns = [])).push(reader.raw(start, reader.pos));
                    }
                }
                if (_end !== $undefined)
                    throw $Error("missing end group");
                return message;
            };

            /**
             * Decodes an Envelope message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof omnimesh.envelope.Envelope
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {omnimesh.envelope.Envelope & omnimesh.envelope.Envelope.$Shape} Envelope
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Envelope.decodeDelimited = function(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };

            /**
             * Verifies an Envelope message.
             * @function verify
             * @memberof omnimesh.envelope.Envelope
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Envelope.verify = function (message, _depth) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (_depth === $undefined)
                    _depth = 0;
                if (_depth > $util.recursionLimit)
                    return "max depth exceeded";
                if (message.id != null && $Object.hasOwnProperty.call(message, "id"))
                    if (!(message.id && typeof message.id.length === "number" || $util.isString(message.id)))
                        return "id: buffer expected";
                if (message.senderPubkey != null && $Object.hasOwnProperty.call(message, "senderPubkey"))
                    if (!(message.senderPubkey && typeof message.senderPubkey.length === "number" || $util.isString(message.senderPubkey)))
                        return "senderPubkey: buffer expected";
                if (message.timestamp != null && $Object.hasOwnProperty.call(message, "timestamp"))
                    if (!$util.isInteger(message.timestamp) && !(message.timestamp && $util.isInteger(message.timestamp.low) && $util.isInteger(message.timestamp.high)))
                        return "timestamp: integer|Long expected";
                if (message.signature != null && $Object.hasOwnProperty.call(message, "signature"))
                    if (!(message.signature && typeof message.signature.length === "number" || $util.isString(message.signature)))
                        return "signature: buffer expected";
                if (message.payloadType != null && $Object.hasOwnProperty.call(message, "payloadType"))
                    switch (message.payloadType) {
                    default:
                        return "payloadType: enum value expected";
                    case 0:
                    case 1:
                    case 2:
                    case 3:
                    case 4:
                        break;
                    }
                if (message.encryptedBody != null && $Object.hasOwnProperty.call(message, "encryptedBody"))
                    if (!(message.encryptedBody && typeof message.encryptedBody.length === "number" || $util.isString(message.encryptedBody)))
                        return "encryptedBody: buffer expected";
                return null;
            };

            /**
             * Creates an Envelope message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof omnimesh.envelope.Envelope
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {omnimesh.envelope.Envelope} Envelope
             */
            Envelope.fromObject = function (object, _depth) {
                if (object instanceof $root.omnimesh.envelope.Envelope)
                    return object;
                if (!$util.isObject(object))
                    throw $TypeError(".omnimesh.envelope.Envelope: object expected");
                if (_depth === $undefined)
                    _depth = 0;
                if (_depth > $util.recursionLimit)
                    throw $Error("max depth exceeded");
                var message = new $root.omnimesh.envelope.Envelope();
                if (object.id != null)
                    if (object.id.length)
                        if (typeof object.id === "string")
                            $util.base64.decode(object.id, message.id = $util.newBuffer($util.base64.length(object.id)), 0);
                        else if (object.id.length >= 0)
                            message.id = object.id;
                if (object.senderPubkey != null)
                    if (object.senderPubkey.length)
                        if (typeof object.senderPubkey === "string")
                            $util.base64.decode(object.senderPubkey, message.senderPubkey = $util.newBuffer($util.base64.length(object.senderPubkey)), 0);
                        else if (object.senderPubkey.length >= 0)
                            message.senderPubkey = object.senderPubkey;
                if (object.timestamp != null)
                    if (typeof object.timestamp === "object" ? object.timestamp.low || object.timestamp.high : $Number(object.timestamp) !== 0)
                        if ($util.Long)
                            message.timestamp = $util.Long.fromValue(object.timestamp, false);
                        else if (typeof object.timestamp === "string")
                            message.timestamp = $parseInt(object.timestamp, 10);
                        else if (typeof object.timestamp === "number")
                            message.timestamp = object.timestamp;
                        else if (typeof object.timestamp === "object")
                            message.timestamp = new $util.LongBits(object.timestamp.low >>> 0, object.timestamp.high >>> 0).toNumber();
                if (object.signature != null)
                    if (object.signature.length)
                        if (typeof object.signature === "string")
                            $util.base64.decode(object.signature, message.signature = $util.newBuffer($util.base64.length(object.signature)), 0);
                        else if (object.signature.length >= 0)
                            message.signature = object.signature;
                if (object.payloadType !== 0 && (typeof object.payloadType !== "string" || $root.omnimesh.envelope.Envelope.Type[object.payloadType] !== 0))
                    switch (object.payloadType) {
                    default:
                        if (typeof object.payloadType === "number") {
                            message.payloadType = object.payloadType;
                            break;
                        }
                        break;
                    case "UNKNOWN":
                    case 0:
                        message.payloadType = 0;
                        break;
                    case "CHAT":
                    case 1:
                        message.payloadType = 1;
                        break;
                    case "MARKET":
                    case 2:
                        message.payloadType = 2;
                        break;
                    case "BLOCKCHAIN":
                    case 3:
                        message.payloadType = 3;
                        break;
                    case "SIGNALING":
                    case 4:
                        message.payloadType = 4;
                        break;
                    }
                if (object.encryptedBody != null)
                    if (object.encryptedBody.length)
                        if (typeof object.encryptedBody === "string")
                            $util.base64.decode(object.encryptedBody, message.encryptedBody = $util.newBuffer($util.base64.length(object.encryptedBody)), 0);
                        else if (object.encryptedBody.length >= 0)
                            message.encryptedBody = object.encryptedBody;
                return message;
            };

            /**
             * Creates a plain object from an Envelope message. Also converts values to other types if specified.
             * @function toObject
             * @memberof omnimesh.envelope.Envelope
             * @static
             * @param {omnimesh.envelope.Envelope} message Envelope
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Envelope.toObject = function (message, options, _depth) {
                if (!options)
                    options = {};
                if (_depth === $undefined)
                    _depth = 0;
                if (_depth > $util.recursionLimit)
                    throw $Error("max depth exceeded");
                var object = {};
                if (options.defaults) {
                    if (options.bytes === $String)
                        object.id = "";
                    else {
                        object.id = [];
                        if (options.bytes !== $Array)
                            object.id = $util.newBuffer(object.id);
                    }
                    if (options.bytes === $String)
                        object.senderPubkey = "";
                    else {
                        object.senderPubkey = [];
                        if (options.bytes !== $Array)
                            object.senderPubkey = $util.newBuffer(object.senderPubkey);
                    }
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.timestamp = options.longs === $String ? long.toString() : options.longs === $Number ? long.toNumber() : typeof $BigInt !== "undefined" && options.longs === $BigInt ? long.toBigInt() : long;
                    } else
                        object.timestamp = options.longs === $String ? "0" : typeof $BigInt !== "undefined" && options.longs === $BigInt ? $BigInt("0") : 0;
                    if (options.bytes === $String)
                        object.signature = "";
                    else {
                        object.signature = [];
                        if (options.bytes !== $Array)
                            object.signature = $util.newBuffer(object.signature);
                    }
                    object.payloadType = options.enums === $String ? "UNKNOWN" : 0;
                    if (options.bytes === $String)
                        object.encryptedBody = "";
                    else {
                        object.encryptedBody = [];
                        if (options.bytes !== $Array)
                            object.encryptedBody = $util.newBuffer(object.encryptedBody);
                    }
                }
                if (message.id != null && $Object.hasOwnProperty.call(message, "id"))
                    object.id = options.bytes === $String ? $util.base64.encode(message.id, 0, message.id.length) : options.bytes === $Array ? $Array.prototype.slice.call(message.id) : message.id;
                if (message.senderPubkey != null && $Object.hasOwnProperty.call(message, "senderPubkey"))
                    object.senderPubkey = options.bytes === $String ? $util.base64.encode(message.senderPubkey, 0, message.senderPubkey.length) : options.bytes === $Array ? $Array.prototype.slice.call(message.senderPubkey) : message.senderPubkey;
                if (message.timestamp != null && $Object.hasOwnProperty.call(message, "timestamp"))
                    if (typeof $BigInt !== "undefined" && options.longs === $BigInt)
                        object.timestamp = typeof message.timestamp === "number" ? $BigInt(message.timestamp) : $util.Long.fromBits(message.timestamp.low >>> 0, message.timestamp.high >>> 0, false).toBigInt();
                    else if (typeof message.timestamp === "number")
                        object.timestamp = options.longs === $String ? $String(message.timestamp) : message.timestamp;
                    else
                        object.timestamp = options.longs === $String ? $util.Long.prototype.toString.call(message.timestamp) : options.longs === $Number ? new $util.LongBits(message.timestamp.low >>> 0, message.timestamp.high >>> 0).toNumber() : message.timestamp;
                if (message.signature != null && $Object.hasOwnProperty.call(message, "signature"))
                    object.signature = options.bytes === $String ? $util.base64.encode(message.signature, 0, message.signature.length) : options.bytes === $Array ? $Array.prototype.slice.call(message.signature) : message.signature;
                if (message.payloadType != null && $Object.hasOwnProperty.call(message, "payloadType"))
                    object.payloadType = options.enums === $String ? $root.omnimesh.envelope.Envelope.Type[message.payloadType] === $undefined ? message.payloadType : $root.omnimesh.envelope.Envelope.Type[message.payloadType] : message.payloadType;
                if (message.encryptedBody != null && $Object.hasOwnProperty.call(message, "encryptedBody"))
                    object.encryptedBody = options.bytes === $String ? $util.base64.encode(message.encryptedBody, 0, message.encryptedBody.length) : options.bytes === $Array ? $Array.prototype.slice.call(message.encryptedBody) : message.encryptedBody;
                return object;
            };

            /**
             * Converts this Envelope to JSON.
             * @function toJSON
             * @memberof omnimesh.envelope.Envelope
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Envelope.prototype.toJSON = function() {
                return Envelope.toObject(this, $protobuf.util.toJSONOptions);
            };

            /**
             * Gets the type url for Envelope
             * @function getTypeUrl
             * @memberof omnimesh.envelope.Envelope
             * @static
             * @param {string} [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
             * @returns {string} The type url
             */
            Envelope.getTypeUrl = function(prefix) {
                if (prefix === $undefined)
                    prefix = "type.googleapis.com";
                return prefix + "/omnimesh.envelope.Envelope";
            };

            /**
             * Type enum.
             * @name omnimesh.envelope.Envelope.Type
             * @enum {number}
             * @property {number} UNKNOWN=0 UNKNOWN value
             * @property {number} CHAT=1 CHAT value
             * @property {number} MARKET=2 MARKET value
             * @property {number} BLOCKCHAIN=3 BLOCKCHAIN value
             * @property {number} SIGNALING=4 SIGNALING value
             */
            Envelope.Type = (function() {
                var valuesById = {}, values = $Object.create(valuesById);
                values[valuesById[0] = "UNKNOWN"] = 0;
                values[valuesById[1] = "CHAT"] = 1;
                values[valuesById[2] = "MARKET"] = 2;
                values[valuesById[3] = "BLOCKCHAIN"] = 3;
                values[valuesById[4] = "SIGNALING"] = 4;
                return values;
            })();

            return Envelope;
        })();

        return envelope;
    })();

    return omnimesh;
})();

module.exports = $root;
