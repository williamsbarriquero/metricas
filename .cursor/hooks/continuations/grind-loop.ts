/**
 * TDD Grinder - Continuation Hook
 * Intercepta stdin do Cursor com contexto de execução (conversation_id, status, loop_count)
 * Verifica exit code de testes e .cursor/scratchpad.md por ALL_TESTS_PASSED: true
 */
import { readFileSync, existsSync } from 'fs';
import { join } from 'path';
import { spawnSync } from 'child_process';

interface CursorContext {
  conversation_id?: string;
  status?: string;
  loop_count?: number;
}

function detectTestCommand(): string | null {
  const cwd = process.cwd();
  if (existsSync(join(cwd, 'package.json'))) return 'npm test';
  if (existsSync(join(cwd, 'go.mod'))) return 'go test ./...';
  if (existsSync(join(cwd, 'gradlew'))) return './gradlew test';
  if (existsSync(join(cwd, 'mvnw'))) return './mvnw test';
  if (existsSync(join(cwd, 'pom.xml'))) return './mvnw test';
  return null;
}

function runTests(): { exitCode: number; command: string } | null {
  const command = detectTestCommand();
  if (!command) return null;

  const result = spawnSync(command, {
    shell: true,
    cwd: process.cwd(),
    encoding: 'utf-8',
    timeout: 120_000,
  });

  return { exitCode: result.status ?? 1, command };
}

async function main() {
  try {
    const stdin = await new Response(Bun.stdin).text();
    const context: CursorContext = stdin ? JSON.parse(stdin) : {};

    if (context.status === 'aborted') {
      console.log(JSON.stringify({}));
      process.exit(0);
    }

    const loopCount = context.loop_count || 1;
    const maxIterations = 5;

    const scratchpadPath = join(process.cwd(), '.cursor', 'scratchpad.md');
    let scratchpadPassed = false;

    try {
      const scratchpad = readFileSync(scratchpadPath, 'utf-8');
      if (scratchpad.includes('ALL_TESTS_PASSED: true')) {
        scratchpadPassed = true;
      }
    } catch {
      // scratchpad não existe
    }

    const testResult = runTests();
    const testExitOk = !testResult || testResult.exitCode === 0;

    // Sucesso somente com testes verdes (ou sem runner) E scratchpad confirmado
    if (scratchpadPassed && testExitOk) {
      console.log(JSON.stringify({}));
      process.exit(0);
    }

    if (loopCount >= maxIterations) {
      console.log(JSON.stringify({
        error: `Limite de ${maxIterations} iterações atingido sem sucesso nos testes.`,
      }));
      process.exit(1);
    }

    const cmdInfo = testResult ? ` Comando: ${testResult.command}, exit ${testResult.exitCode}.` : '';
    let followupMessage: string;

    if (testResult && testResult.exitCode === 0 && !scratchpadPassed) {
      followupMessage =
        `Testes passaram (${testResult.command}, exit 0), mas ALL_TESTS_PASSED não é true no scratchpad. Atualize .cursor/scratchpad.md com ALL_TESTS_PASSED: true antes de encerrar.`;
    } else {
      followupMessage =
        `Testes falharam ou scratchpad incompleto.${cmdInfo} Corrija implementação, execute testes e atualize ALL_TESTS_PASSED no scratchpad.`;
    }

    if (loopCount >= 3) {
      followupMessage =
        `Múltiplas iterações sem sucesso (≥3). Revise a abordagem, consulte diagnósticos LSP e corrija testes/implementação. ${followupMessage}`;
    }

    console.log(JSON.stringify({ followup_message: followupMessage }));
    process.exit(0);
  } catch (error) {
    console.error('Erro no grind-loop:', error);
    process.exit(1);
  }
}

main();
