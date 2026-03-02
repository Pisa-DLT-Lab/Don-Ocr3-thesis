const hre = require("hardhat");
require("dotenv").config({ path: "../.env" });

async function main() {
  const CONTRACT_ADDR = process.env.VERIFIER_ADDRESS; // Assicurati che nel .env ci sia l'indirizzo del Verifier nuovo

  console.log(`\nVerifying Oracle Data on Chain`);
  console.log(`Target Contract: ${CONTRACT_ADDR}`);

  // Collegamento al contratto
  const verifier = await hre.ethers.getContractAt("OracleVerifier", CONTRACT_ADDR);

  try {
    // 1. TROVARE L'ID: Cerchiamo negli eventi passati per trovare l'ultimo salvataggio
    console.log("   -> Searching for 'JobCompleted' events...");
    
    // Crea il filtro per l'evento
    const filter = verifier.filters.JobCompleted();
    // Ottieni lo storico degli eventi (dal blocco genesi a oggi)
    const events = await verifier.queryFilter(filter);

    if (events.length === 0) {
        console.log("No outcomes found on chain yet.");
        return;
    }

    // Prendiamo l'ultimo evento emesso (il più recente)
    const latestEvent = events[events.length - 1];
    const requestId = latestEvent.args[0]; // Il primo argomento dell'evento è requestId
    const submitterFromEvent = latestEvent.args[1];
    const lengthFromEvent = latestEvent.args[2];

    console.log(`\nFound latest Job ID: #${requestId}`);
    console.log(`   Submitter: ${submitterFromEvent}`);
    console.log(`   Declared Length: ${lengthFromEvent}`);

    // 2. LEGGERE I DATI: Chiamiamo la funzione getResult con l'ID trovato
    console.log(`\nFetching data from Storage (getResult)...`);
    
    // Restituisce: [flatMatrix, submitter, timestamp]
    const result = await verifier.getResult(requestId);
    
    const flatMatrix = result[0]; // Array di BigInt
    const submitter = result[1];
    const timestamp = result[2];

    // 3. CONVERSIONE E VISUALIZZAZIONE
    console.log("\nData Analysis:");
    console.log(`   - Timestamp: ${new Date(Number(timestamp) * 1000).toLocaleString()}`);
    console.log(`   - Saved By:  ${submitter}`);
    console.log(`   - Array Len: ${flatMatrix.length} items`);

    console.log("\nDecoded Values (First 10 items):");
    console.log("   (Converting fixed-point 1e18 to Float)");
    console.log("   ----------------------------------------");

    // Mostriamo solo i primi 10 per non intasare la console se sono 2500
    const limit = Math.min(flatMatrix.length, 10);
    
    for (let i = 0; i < limit; i++) {
        // I dati arrivano come BigInt (es. 1000000000000000000n)
        // Usiamo formatUnits per rimettere la virgola (divide per 10^18)
        const floatValue = hre.ethers.formatUnits(flatMatrix[i], 18);
        console.log(`   [${i}]: ${floatValue}`);
    }

    if (flatMatrix.length > 10) {
        console.log(`   ... (+ other ${flatMatrix.length - 10} items hidden)`);
        
        // Stampiamo l'ultimo per conferma
        const lastVal = hre.ethers.formatUnits(flatMatrix[flatMatrix.length - 1], 18);
        console.log(`   [LAST]: ${lastVal}`);
    }

    console.log("\nSUCCESS: Data verified on-chain!");

  } catch (error) {
    console.error("\nError:");
    // Se l'errore è "Result not found", vuol dire che l'ID non esiste o saved=false
    if (error.reason) console.error("Reason:", error.reason);
    console.error(error);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});