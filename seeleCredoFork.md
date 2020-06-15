1. FORKHEIGHT define

2. fork genesis 
   (if use the old genesis module, will remove this part verification, is it ok?)
   2.1 create
   2.2 verfication


3. mining : task
    (3.1) if height < FORKHEIGHT: create new task after FORKHEIGHT height;
            (3.1.1) create task with parent (modified)
            (3.1.2) saveBlock (mined or sychronized)
    (3.2) if height > FORKHEIGHT: delete and reverse, and then back to step 3.1;

    (engine verification: verifyHeaderCommon and verifyHeader)

4. sychronize (BE CAREFUL ABOUT THE DIFFICULTY!)
   
    (4.0) check the peer's fork gensis hash is right, if and only if yes, then

    (4.1) height > FORKHEIGHT and difficulty is bigger;
            GIVE FORKGENESIS A HIGHEST PRIORITY
            (4.1.1) first check FORKHEIGHT hash = FORKHEIGHT_HASH: if yes, node already start from fork, sync;
                                                                    if no, reverse and sync;  
    (4.2) height < FORKHEIGTH 
            Sync from FORKGENESIS, need to handle discrete database!

5. p2p: peer selection:
   check the sync will delete peer. modify the case!

6. new coinbase CAN NOT height before FORKHEIGHT