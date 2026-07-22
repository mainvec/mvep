import { describe, it, expect } from 'vitest';
import { BasePackage, SimplePackage, createCommand } from '../package';

describe('package', () => {
  describe('SimplePackage', () => {
    it('creates a package with name and commands', () => {
      const pkg = new SimplePackage('myservice', ['GetUser', 'CreateUser']);

      expect(pkg.getName()).toBe('myservice');
    });

    it('returns instances for registered commands', () => {
      const pkg = new SimplePackage('myservice', ['GetUser']);

      const instance = pkg.instanceOf('GetUser');
      expect(instance).toBeDefined();
      expect((instance as any)._cmdName).toBe('GetUser');

      // Also registers result type
      const resultInstance = pkg.instanceOf('GetUserResult');
      expect(resultInstance).toBeDefined();
    });

    it('returns undefined for unknown commands', () => {
      const pkg = new SimplePackage('myservice', ['GetUser']);

      const instance = pkg.instanceOf('UnknownCmd');
      expect(instance).toBeUndefined();
    });

    it('returns command name from instance', () => {
      const pkg = new SimplePackage('myservice', ['GetUser']);

      const cmd = { _cmdName: 'GetUser', userId: 123 };
      expect(pkg.nameOf(cmd)).toBe('GetUser');
    });

    it('returns empty string for unregistered command', () => {
      const pkg = new SimplePackage('myservice', ['GetUser']);

      const cmd = { _cmdName: 'Unknown' };
      expect(pkg.nameOf(cmd)).toBe('');
    });

    it('allows adding commands after creation', () => {
      const pkg = new SimplePackage('myservice');
      pkg.addCommand('NewCommand');

      const instance = pkg.instanceOf('NewCommand');
      expect(instance).toBeDefined();
    });
  });

  describe('BasePackage', () => {
    it('creates a package with name', () => {
      const pkg = new BasePackage('myservice');
      expect(pkg.getName()).toBe('myservice');
    });

    it('registers commands with factories', () => {
      const pkg = new BasePackage('myservice')
        .registerCommand({
          name: 'GetUser',
          createCmd: () => ({ userId: 0 }),
          createResult: () => ({ user: null }),
        });

      const cmdInstance = pkg.instanceOf('GetUser');
      expect(cmdInstance).toEqual({ userId: 0 });

      const resultInstance = pkg.instanceOf('GetUserResult');
      expect(resultInstance).toEqual({ user: null });
    });

    it('registers command classes', () => {
      class GetUserCmd {
        userId = 0;
      }
      class GetUserResult {
        user: any = null;
      }

      const pkg = new BasePackage('myservice')
        .registerCommandClass('GetUser', GetUserCmd, GetUserResult);

      const cmdInstance = pkg.instanceOf('GetUser');
      expect(cmdInstance).toBeInstanceOf(GetUserCmd);

      const resultInstance = pkg.instanceOf('GetUserResult');
      expect(resultInstance).toBeInstanceOf(GetUserResult);
    });

    it('returns command name from class instance', () => {
      class GetUserCmd {
        userId = 0;
      }
      class GetUserResult {
        user: any = null;
      }

      const pkg = new BasePackage('myservice')
        .registerCommandClass('GetUser', GetUserCmd, GetUserResult);

      const cmd = new GetUserCmd();
      expect(pkg.nameOf(cmd)).toBe('GetUser');
    });

    it('returns empty string for unknown type', () => {
      const pkg = new BasePackage('myservice');
      expect(pkg.nameOf({ foo: 'bar' })).toBe('');
      expect(pkg.nameOf(null)).toBe('');
      expect(pkg.nameOf(undefined)).toBe('');
    });
  });

  describe('createCommand', () => {
    it('creates command object with _cmdName', () => {
      const cmd = createCommand('GetUser', { userId: 123 });

      expect(cmd._cmdName).toBe('GetUser');
      expect(cmd.userId).toBe(123);
    });

    it('works with SimplePackage', () => {
      const pkg = new SimplePackage('myservice', ['GetUser']);
      const cmd = createCommand('GetUser', { userId: 123 });

      expect(pkg.nameOf(cmd)).toBe('GetUser');
    });
  });
});
