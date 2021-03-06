import { ApolloError } from "@apollo/client";
import { GraphQLError } from "graphql";

import { APIValidationError } from "./validation";
import { APIInvariantViolationError } from "./invariant";
import { APIPasswordPolicyViolatedError } from "./password";
import { APIForbiddenError } from "./forbidden";
import {
  APIDuplicatedDomainError,
  APIDomainVerifiedError,
  APIDomainNotFoundError,
  APIDomainNotCustomError,
  APIDomainVerificationFailedError,
  APIInvalidDomainError,
} from "./domain";
import {
  APIDuplicatedCollaboratorInvitationError,
  APICollaboratorSelfDeletionError,
  APICollaboratorDuplicateError,
  APICollaboratorInvitationInvalidCodeError,
  APICollaboratorInvitationInvalidEmailError,
} from "./collaborator";
import { APIDuplicatedAppIDError } from "./apps";
import { APIResourceNotFoundError } from "./resources";

export type APIError =
  | APIValidationError
  | APIInvariantViolationError
  | APIPasswordPolicyViolatedError
  | APIForbiddenError
  | APIDuplicatedDomainError
  | APIDomainVerifiedError
  | APIDomainNotFoundError
  | APIDomainNotCustomError
  | APIDomainVerificationFailedError
  | APIInvalidDomainError
  | APIDuplicatedCollaboratorInvitationError
  | APICollaboratorSelfDeletionError
  | APICollaboratorInvitationInvalidCodeError
  | APICollaboratorInvitationInvalidEmailError
  | APICollaboratorDuplicateError
  | APIDuplicatedAppIDError
  | APIResourceNotFoundError;

export function isAPIError(value?: { [key: string]: any }): value is APIError {
  if (value == null) {
    return false;
  }
  return "errorName" in value && "reason" in value;
}

export function isApolloError(error: unknown): error is ApolloError {
  return error instanceof ApolloError;
}

export function extractAPIError(error: GraphQLError): APIError | undefined {
  if (isAPIError(error.extensions)) {
    return error.extensions;
  }
  return undefined;
}
