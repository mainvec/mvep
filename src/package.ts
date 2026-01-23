/**
 * Package interface and utilities for MVP
 */

/**
 * Package defines the interface for MVP packages
 * A package groups related commands together
 */
export interface Package {
  /**
   * Returns the package name
   */
  getName(): string;

  /**
   * Creates a new instance of a command by name
   * Returns undefined if the command is not found
   */
  instanceOf(cmdName: string): unknown | undefined;

  /**
   * Returns the command name for a given command instance
   * Returns empty string if the command is not registered
   */
  nameOf(cmd: unknown): string;
}

/**
 * CommandDefinition defines a command and its result type
 */
export interface CommandDefinition<TCmd = unknown, TResult = unknown> {
  /** Command name */
  name: string;
  /** Factory function to create a new command instance */
  createCmd?: () => TCmd;
  /** Factory function to create a new result instance */
  createResult?: () => TResult;
}

/**
 * BasePackage provides a basic implementation of the Package interface
 */
export class BasePackage implements Package {
  private name: string;
  private commands: Map<string, CommandDefinition>;
  private typeToName: Map<Function, string>;

  constructor(name: string) {
    this.name = name;
    this.commands = new Map();
    this.typeToName = new Map();
  }

  /**
   * Registers a command with the package
   */
  registerCommand<TCmd, TResult>(def: CommandDefinition<TCmd, TResult>): this {
    this.commands.set(def.name, def);
    this.commands.set(def.name + 'Result', {
      name: def.name + 'Result',
      createCmd: def.createResult,
    });
    return this;
  }

  /**
   * Registers a command with class constructors
   */
  registerCommandClass<TCmd extends new () => unknown, TResult extends new () => unknown>(
    name: string,
    cmdClass: TCmd,
    resultClass: TResult
  ): this {
    this.commands.set(name, {
      name,
      createCmd: () => new cmdClass(),
      createResult: () => new resultClass(),
    });
    this.commands.set(name + 'Result', {
      name: name + 'Result',
      createCmd: () => new resultClass(),
    });
    this.typeToName.set(cmdClass, name);
    this.typeToName.set(resultClass, name + 'Result');
    return this;
  }

  getName(): string {
    return this.name;
  }

  instanceOf(cmdName: string): unknown | undefined {
    const def = this.commands.get(cmdName);
    if (def?.createCmd) {
      return def.createCmd();
    }
    // Return an empty object if no factory is defined
    return def ? {} : undefined;
  }

  nameOf(cmd: unknown): string {
    if (cmd === null || cmd === undefined) {
      return '';
    }

    // Check by constructor
    const constructor = (cmd as object).constructor;
    if (constructor && this.typeToName.has(constructor)) {
      return this.typeToName.get(constructor) || '';
    }

    // Check if cmd has a _cmdName property
    if (typeof cmd === 'object' && '_cmdName' in cmd) {
      return (cmd as { _cmdName: string })._cmdName;
    }

    return '';
  }
}

/**
 * SimplePackage is a package implementation that uses plain objects
 * Command names are derived from the object's _cmdName property or provided explicitly
 */
export class SimplePackage implements Package {
  private name: string;
  private commands: Set<string>;

  constructor(name: string, commands: string[] = []) {
    this.name = name;
    this.commands = new Set(commands);
    // Also register result types
    for (const cmd of commands) {
      this.commands.add(cmd + 'Result');
    }
  }

  /**
   * Adds a command name to the package
   */
  addCommand(cmdName: string): this {
    this.commands.add(cmdName);
    this.commands.add(cmdName + 'Result');
    return this;
  }

  getName(): string {
    return this.name;
  }

  instanceOf(cmdName: string): unknown | undefined {
    if (this.commands.has(cmdName)) {
      return { _cmdName: cmdName };
    }
    return undefined;
  }

  nameOf(cmd: unknown): string {
    if (cmd === null || cmd === undefined) {
      return '';
    }

    if (typeof cmd === 'object' && '_cmdName' in cmd) {
      const cmdName = (cmd as { _cmdName: string })._cmdName;
      if (this.commands.has(cmdName)) {
        return cmdName;
      }
    }

    return '';
  }
}

/**
 * Creates a command object with a name
 */
export function createCommand<T extends object>(name: string, data: T): T & { _cmdName: string } {
  return { ...data, _cmdName: name };
}
