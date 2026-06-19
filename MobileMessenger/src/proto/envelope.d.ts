import * as $protobuf from "protobufjs";
import Long = require("long");

/** Namespace omnimesh. */
export namespace omnimesh {

    /** Namespace envelope. */
    namespace envelope {

        /**
         * Properties of an Envelope.
         * @deprecated Use omnimesh.envelope.Envelope.$Properties instead.
         */
        interface IEnvelope extends omnimesh.envelope.Envelope.$Properties {
        }

        /** Represents an Envelope. */
        class Envelope {

            /**
             * Constructs a new Envelope.
             * @param [properties] Properties to set
             */
            constructor(properties?: omnimesh.envelope.Envelope.$Properties);

            /** Unknown fields preserved while decoding when enabled */
            $unknowns?: Uint8Array[];

            /** Envelope id. */
            id: Uint8Array;

            /** Envelope senderPubkey. */
            senderPubkey: Uint8Array;

            /** Envelope timestamp. */
            timestamp: (number|Long);

            /** Envelope signature. */
            signature: Uint8Array;

            /** Envelope payloadType. */
            payloadType: omnimesh.envelope.Envelope.Type;

            /** Envelope encryptedBody. */
            encryptedBody: Uint8Array;

            /**
             * Creates a new Envelope instance using the specified properties.
             * @param [properties] Properties to set
             * @returns Envelope instance
             */
            static create(properties: omnimesh.envelope.Envelope.$Shape): omnimesh.envelope.Envelope & omnimesh.envelope.Envelope.$Shape;
            static create(properties?: omnimesh.envelope.Envelope.$Properties): omnimesh.envelope.Envelope;

            /**
             * Encodes the specified Envelope message. Does not implicitly {@link omnimesh.envelope.Envelope.verify|verify} messages.
             * @param message Envelope message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encode(message: omnimesh.envelope.Envelope.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Encodes the specified Envelope message, length delimited. Does not implicitly {@link omnimesh.envelope.Envelope.verify|verify} messages.
             * @param message Envelope message or plain object to encode
             * @param [writer] Writer to encode to
             * @returns Writer
             */
            static encodeDelimited(message: omnimesh.envelope.Envelope.$Properties, writer?: $protobuf.Writer): $protobuf.Writer;

            /**
             * Decodes an Envelope message from the specified reader or buffer.
             * @param reader Reader or buffer to decode from
             * @param [length] Message length if known beforehand
             * @returns {omnimesh.envelope.Envelope & omnimesh.envelope.Envelope.$Shape} Envelope
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): omnimesh.envelope.Envelope & omnimesh.envelope.Envelope.$Shape;

            /**
             * Decodes an Envelope message from the specified reader or buffer, length delimited.
             * @param reader Reader or buffer to decode from
             * @returns {omnimesh.envelope.Envelope & omnimesh.envelope.Envelope.$Shape} Envelope
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): omnimesh.envelope.Envelope & omnimesh.envelope.Envelope.$Shape;

            /**
             * Verifies an Envelope message.
             * @param message Plain object to verify
             * @returns `null` if valid, otherwise the reason why it is not
             */
            static verify(message: { [k: string]: any }): (string|null);

            /**
             * Creates an Envelope message from a plain object. Also converts values to their respective internal types.
             * @param object Plain object
             * @returns Envelope
             */
            static fromObject(object: { [k: string]: any }): omnimesh.envelope.Envelope;

            /**
             * Creates a plain object from an Envelope message. Also converts values to other types if specified.
             * @param message Envelope
             * @param [options] Conversion options
             * @returns Plain object
             */
            static toObject(message: omnimesh.envelope.Envelope, options?: $protobuf.IConversionOptions): { [k: string]: any };

            /**
             * Converts this Envelope to JSON.
             * @returns JSON object
             */
            toJSON(): { [k: string]: any };

            /**
             * Gets the type url for Envelope
             * @param [prefix] Custom type url prefix, defaults to `"type.googleapis.com"`
             * @returns The type url
             */
            static getTypeUrl(prefix?: string): string;
        }

        namespace Envelope {

            /** Properties of an Envelope. */
            interface $Properties {

                /** Envelope id */
                id?: (Uint8Array|null);

                /** Envelope senderPubkey */
                senderPubkey?: (Uint8Array|null);

                /** Envelope timestamp */
                timestamp?: (number|Long|null);

                /** Envelope signature */
                signature?: (Uint8Array|null);

                /** Envelope payloadType */
                payloadType?: (omnimesh.envelope.Envelope.Type|null);

                /** Envelope encryptedBody */
                encryptedBody?: (Uint8Array|null);

                /** Unknown fields preserved while decoding when enabled */
                $unknowns?: Uint8Array[];
            }

            /** Shape of an Envelope. */
            type $Shape = omnimesh.envelope.Envelope.$Properties;

            /** Type enum. */
            enum Type {

                /** UNKNOWN value */
                UNKNOWN = 0,

                /** CHAT value */
                CHAT = 1,

                /** MARKET value */
                MARKET = 2,

                /** BLOCKCHAIN value */
                BLOCKCHAIN = 3,

                /** SIGNALING value */
                SIGNALING = 4
            }
        }
    }
}
